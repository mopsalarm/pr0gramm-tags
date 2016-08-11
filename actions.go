package main

import (
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
)

type Locker struct {
	lock sync.RWMutex
}

func (l *Locker) WithReadLock(fn func()) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	fn()
}

func (l *Locker) WithWriteLock(fn func()) {
	l.lock.Lock()
	defer l.lock.Unlock()
	fn()
}

type storeActions struct {
	Locker
	updateLock sync.Mutex
	store      IterStore
	storeState StoreState
}

func (sa *storeActions) UpdateOnce(db *sqlx.DB) bool {
	log.Debug("Looking for updates...")

	var currentStoreState StoreState
	sa.WithReadLock(func() {
		currentStoreState = sa.storeState
	})

	updates, newState, more := FetchUpdates(db, currentStoreState)
	log.WithField("keyCount", updates.KeyCount()).Debug("Will merge updates now")

	// allow only one update at a time
	sa.updateLock.Lock()
	defer sa.updateLock.Unlock()

	// now merge the updates into the store...
	start := time.Now()
	MergeIterStores(sa.store, updates, &sa.lock)

	sa.WithWriteLock(func() {
		sa.storeState = newState

		log.WithField("duration", time.Since(start)).
			WithField("keyCount", updates.KeyCount()).
			WithField("state", sa.storeState).
			WithField("memory", sa.store.MemorySize()).
			Info("Update finsihed")
	})

	return more
}

func (sa *storeActions) WriteCheckpoint(file string) (err error) {
	sa.WithReadLock(func() {
		start := time.Now()
		err = WriteCheckpointFile(file, sa.storeState, sa.store)
		if err != nil {
			log.Println("Could not write checkpoint file:", err)
		}

		log.Println("Writing checkpoint took", time.Since(start))
	})

	return
}

func (sa *storeActions) Search(query string) (result []int32, err error) {
	err = withRecovery("search", func() {
		queryLowerCase := strings.ToLower(query)

		sa.WithReadLock(func() {
			parser := NewParser(strings.NewReader(queryLowerCase), func(str string) ItemIterator {
				var hash uint32
				if str != "__all" {
					if len(str) < 2 || str[1] != ':' {
						str = CleanString(str)
					}

					hash = HashWord(str)
				}

				return sa.store.GetIterator(hash)
			})

			iter := NewNegateIterator(NewLimitIterator(120, parser.Parse()))
			result = IteratorToList(nil, iter)
		})
	})

	return
}
