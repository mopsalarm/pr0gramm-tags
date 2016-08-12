package main

import (
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	"github.com/mopsalarm/go-pr0gramm-tags/store"
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
	updateLock sync.Mutex
	store      store.IterStore
	storeState store.StoreState
}

func (sa *storeActions) UpdateOnce(db *sqlx.DB) bool {
	log.Debug("Looking for updates...")

	var currentStoreState store.StoreState
	sa.WithReadLock(func() {
		currentStoreState = sa.storeState
	})

	updates, newState, more := FetchUpdates(db, currentStoreState)
	log.WithField("keyCount", updates.KeyCount()).Debug("Will merge updates now")

	// allow only one update at a time
	withLock(&sa.updateLock, func() {
		metricsKeysCount.Update(int64(sa.store.KeyCount()))
		metricsUpdaterKeysChanged.Inc(int64(updates.KeyCount()))

		// now merge the updates into the store...
		start := time.Now()

		// get the keys while holding the lock
		for _, key := range updates.Keys() {
			values := store.IteratorToList(nil, store.NewOrIterator(sa.store.GetIterator(key), updates.GetIterator(key)))

			sa.WithWriteLock(func() {
				sa.store.Replace(key, values)
			})
		}

		sa.WithWriteLock(func() {
			sa.storeState = newState

			log.WithField("duration", time.Since(start)).
				WithField("keyCount", updates.KeyCount()).
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

func (sa *storeActions) Search(query string, olderThan int32) (result []int32, err error) {
	err = withRecovery("search", func() {
		metricsSearch.Time(func() {
			queryLowerCase := strings.ToLower(query)
			sa.WithReadLock(func() {
				log.WithField("query", query).WithField("older", olderThan).Debug("Start search query")

				parser := NewParser(strings.NewReader(queryLowerCase), func(str string) store.ItemIterator {
					var hash uint32
					if str != "__all" {
						if len(str) < 2 || str[1] != ':' {
							str = CleanString(str)
						}

						hash = HashWord(str)
					}

					return sa.store.GetIterator(hash)
				})

				iter := parser.Parse()
				if olderThan > 0 {
					// skipping posts. we need to invert the item id here, cause
					// the search is running on negative ids internally
					store.IteratorSkipUntil(iter, -olderThan)
				}

				// get the first 120 results
				iter = store.NewLimitIterator(120,  store.NewNegateIterator(iter))
				result = store.IteratorToList(nil, iter)
			})
		})
	})

	return
}
