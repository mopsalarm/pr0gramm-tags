package main

import (
	"sync"
	"time"

	"github.com/cznic/sortutil"
	"github.com/jmoiron/sqlx"
	"github.com/mopsalarm/go-pr0gramm-tags/parser"
	"github.com/mopsalarm/go-pr0gramm-tags/store"
	log "github.com/sirupsen/logrus"
	"strings"
)

type Locker struct {
	lock sync.RWMutex
}

func (l *Locker) WithReadLock(fn func()) {
	withLock(l.lock.RLocker(), func() {
		metricsReadLock.Time(fn)
	})
}

func (l *Locker) WithWriteLock(fn func()) {
	withLock(&l.lock, func() {
		metricsWriteLock.Time(fn)
	})
}

func withLock(locker sync.Locker, fn func()) {
	locker.Lock()
	defer locker.Unlock()

	fn()
}

type storeActions struct {
	Locker
	UseOptimizer bool
	updateLock   sync.Mutex
	store        store.IterStore
	storeState   store.StoreState
}

func (sa *storeActions) UpdateOnce(db *sqlx.DB) bool {
	log.Debug("Looking for updates...")

	var currentStoreState store.StoreState
	sa.WithReadLock(func() {
		currentStoreState = sa.storeState
	})

	queryStart := time.Now()
	updates, newState, more := FetchUpdates(db, currentStoreState)
	log.WithField("duration", time.Since(queryStart)).Debug("Looking for new updates finished")

	// allow only one update at a time
	withLock(&sa.updateLock, func() {
		log.WithField("keyCount", updates.KeyCount()).Debug("Will merge updates now")
		metricsKeysCount.Update(int64(sa.store.KeyCount()))

		// now merge the updates into the store...
		start := time.Now()

		// get the keys while holding the lock
		changedKeyCount := int64(0)
		for _, key := range updates.Keys() {
			// get the values from both iterators
			storeValues := store.IteratorToList(nil, sa.store.GetIterator(key))
			updateValues := store.IteratorToList(nil, updates.GetIterator(key))

			// an update is only required, if one of the updated values are not in the old values.
			requireUpdate := false
			notFound := len(storeValues)
			for _, value := range updateValues {
				idx := sortutil.SearchInt32s(storeValues, value)
				if idx == notFound || storeValues[idx] != value {
					requireUpdate = true
					break
				}
			}

			if requireUpdate {
				changedKeyCount += 1

				values := store.IteratorToList(nil, store.NewOrIterator(
					store.NewSliceIterator(storeValues), store.NewSliceIterator(updateValues)))

				sa.WithWriteLock(func() {
					sa.store.Replace(key, values)
				})
			}
		}

		if changedKeyCount == 0 {
			log.Debug("No updates were merged, state is up-to-date.")
		}

		metricsUpdaterKeysChanged.Inc(changedKeyCount)
		sa.WithWriteLock(func() {
			sa.storeState = newState

			log.WithField("duration", time.Since(start)).
				WithField("keyCount", changedKeyCount).
				WithField("state", sa.storeState).
				WithField("memory", sa.store.MemorySize()).
				Info("Update finished and merged")
		})
	})

	return more
}

func (sa *storeActions) WriteCheckpoint(file string) (err error) {
	sa.WithReadLock(func() {
		start := time.Now()
		err = store.WriteCheckpointFile(file, sa.storeState, sa.store)
		if err != nil {
			log.Warn("Could not write checkpoint file:", err)
			metricsCheckpointError.Inc(1)
		}

		log.WithField("duration", time.Since(start)).Info("Writing checkpoint finished")
	})

	return
}

func (sa *storeActions) Search(query string, olderThan int32, shuffle bool) (result []int32, err error) {
	err = withRecovery("search", func() {
		queryLowerCase := strings.ToLower(query)

		// parse the query into an ast
		pr := parser.NewParser(strings.NewReader(queryLowerCase))
		ast, err := pr.Parse()
		if err != nil {
			panic(err)
		}

		if sa.UseOptimizer {
			// optimize the ast for maximum performance!!!1
			ast = parser.Optimize(ast)
		}

		metricsSearch.Time(func() {
			sa.WithReadLock(func() {
				log.WithField("query", query).WithField("older", olderThan).Debug("Start search query")
				iter := parser.ToIterator(ast, func(str string) store.ItemIterator {
					var hash uint32
					if str != "__all" {
						if len(str) < 2 || str[1] != ':' {
							str = CleanString(str)
						}

						hash = HashWord(str)
					}

					return sa.store.GetIterator(hash)
				})

				switch {
				case shuffle:
					iter = store.NewShuffledIterator(iter)

				case olderThan > 0:
					// skipping posts. we need to invert the item id here, cause
					// the search is running on negative ids internally
					store.IteratorSkipUntil(iter, -olderThan)
				}

				// get the first 120 results
				iter = store.NewLimitIterator(120, store.NewNegateIterator(iter))
				result = store.IteratorToList(nil, iter)
			})
		})
	})

	return
}
