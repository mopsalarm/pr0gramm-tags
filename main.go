package main

import (
	"fmt"
	"os"
	"strings"
	"runtime"
	"hash/fnv"
	"github.com/cznic/sortutil"
	"gopkg.in/cheggaaa/pb.v1"
	"io"
	"sort"
	"github.com/gin-gonic/gin"
	"time"
	"net/http"
	"github.com/jmoiron/sqlx"
	"log"

	_ "github.com/lib/pq"
	"sync"
	"github.com/robfig/cron"
	"sync/atomic"
	"github.com/jessevdk/go-flags"
	"runtime/pprof"
	"math"
)

var umlautReplacer = strings.NewReplacer("ä", "ae", "ü", "ue", "ö", "oe", "ß", "ss", "-", " ")

func CleanString(str string) string {
	return strings.Map(func(c rune) rune {
		if 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == ' ' {
			return c
		} else {
			return -1
		}
	}, umlautReplacer.Replace(str))
}

func ExtractWordsSimple(line string) []string {
	return strings.FieldsFunc(line, func(ch rune) bool {
		return ch == ','
	})
}

func ExtractWords(str string) []string {
	words := strings.Fields(CleanString(str))

	// deduplicate tags.
	n := sortutil.Dedupe(sort.StringSlice(words))
	return words[:n]
}

func HashWord(word string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(word))
	return h.Sum32()
}

func OpenWithProgress(filename string) (io.ReadCloser, *pb.ProgressBar, error) {
	fp, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}

	if stat, err := fp.Stat(); err == nil {
		bar := pb.New64(stat.Size())
		bar.SetUnits(pb.U_BYTES)
		bar.Start()
		return bar.NewProxyReader(fp), bar, nil
	}

	return fp, nil, nil
}

type StoreBuilder struct {
	ShowProgress bool
	byteStore    ByteStore
	iterStore    IterStore
}

func NewStoreBuilder() *StoreBuilder {
	byteStore := NewCppStore()
	return &StoreBuilder{
		byteStore: byteStore,
		// iterStore: &UncompressedStore{byteStore},
		iterStore: &VarintStore{byteStore},
	}
}

func (sb *StoreBuilder) Push(word string, itemId int32) {
	hash := HashWord(word)

	// check if the hash is alredy known before adding it
	known := sb.byteStore.Contains(hash)

	sb.iterStore.PushInt(hash, int32(itemId))

	if ! known {
		// add item virtual tag "__all"
		sb.iterStore.PushInt(0, int32(itemId))
	}
}

func (sb *StoreBuilder) Build() IterStore {
	var bar *pb.ProgressBar
	if sb.ShowProgress {
		bar = pb.StartNew(int(sb.iterStore.KeyCount()))
		bar.ShowFinalTime = true
		defer bar.Finish()
	}

	optimizedStore := &VarintStore{sb.byteStore}
	// optimizedStore := &UncompressedStore{sb.byteStore}
	for _, key := range sb.iterStore.Keys() {
		if bar != nil {
			bar.Increment()
		}

		// get the list of items
		items := IteratorToList(nil, sb.iterStore.GetIterator(key), math.MaxInt32)
		n := sortutil.Dedupe(sortutil.Int32Slice(items))

		// empty the original entry
		sb.byteStore.Clear(key)

		// and optimize it
		optimizedStore.PushInts(key, items[:n])
		optimizedStore.Compact(key)
	}

	return optimizedStore
}

func MergeIterStores(target, other IterStore) {
	for _, key := range other.Keys() {
		values := IteratorToList(nil, NewOrIterator(
			target.GetIterator(key),
			other.GetIterator(key)), 120)

		target.Clear(key)
		target.PushInts(key, values)
	}
}

type TagConsumer func(tagId, itemId int32, tag string)

func StreamTagsFromPostgres(db *sqlx.DB, firstTagId int32, count int, consumer TagConsumer) error {
	rows, err := db.Query(
		"SELECT id, item_id, lower(tag) FROM tags WHERE id >= $1 ORDER BY id ASC LIMIT $2",
		firstTagId, count)

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var tagId, itemId int32
		var tag string

		if err := rows.Scan(&tagId, &itemId, &tag); err != nil {
			return err
		}

		consumer(tagId, itemId, tag)
	}

	return rows.Err()
}

type StoreState struct {
	LastTagId  int32
	LastItemId int32
}

func FetchUpdates(db *sqlx.DB, state StoreState) (IterStore, StoreState, bool) {
	builder := NewStoreBuilder()

	var postInfos []struct {
		Id       int32 `db:"id"`
		Flags    int32 `db:"flags"`
		Score    int32 `db:"score"`
		Promoted bool `db:"promoted"`
		Username string `db:"username"`
		HasText  bool `db:"has_text"`
	}

	err := db.Select(&postInfos, `
		SELECT
			items.id,
			items.flags,
			items.up - items.down as score,
			items.promoted != 0 as promoted,
			lower(items.username) AS username,
			COALESCE(texts.has_text, FALSE) AS has_text
		FROM
			items
			LEFT JOIN items_text texts ON (items.id = texts.item_id)
		WHERE id >= $1
		ORDER BY items.id ASC LIMIT 10000`, state.LastItemId)

	if err != nil {
		log.Println("Error while getting recent posts", err)
	} else {
		for _, postInfo := range postInfos {
			itemId := -postInfo.Id

			builder.Push("u:" + CleanString(postInfo.Username), itemId)

			switch {
			case postInfo.Flags & 1 != 0:
				builder.Push("f:sfw", itemId)
			case postInfo.Flags & 2 != 0:
				builder.Push("f:nsfw", itemId)
			case postInfo.Flags & 4 != 0:
				builder.Push("f:nsfl", itemId)
			}

			if postInfo.Promoted {
				builder.Push("f:top", itemId)
			}

			if postInfo.HasText {
				builder.Push("f:text", itemId)
			}

			// sort posts into bins (size 500) by score.
			// a post with score 1100 will be put into bins 500 and 1000
			for bin := int32(1); bin <= postInfo.Score / 500; bin++ {
				label := fmt.Sprintf("s:%d", (500 * bin))
				builder.Push(label, itemId)
			}

			state.LastItemId = postInfo.Id
		}
	}

	tagCount := 100000
	err = StreamTagsFromPostgres(db, state.LastTagId, tagCount, func(tagId, itemId int32, tag string) {
		for _, word := range ExtractWords(tag) {
			builder.Push(word, -itemId)
		}

		tagCount--
		state.LastTagId = tagId
	});

	if err != nil {
		log.Println("Error while streaming from postgres", err)
	}

	expectMore := tagCount == 0
	return builder.Build(), state, expectMore
}

func PreventConcurrency(fn func()) func() {
	var guard int32
	return func() {
		if atomic.CompareAndSwapInt32(&guard, 0, 1) {
			defer atomic.StoreInt32(&guard, 0)
			fn()
		}
	}
}

func main() {
	var opts struct {
		RebuildItems   bool `long:"rebuild-items" description:"Rescans all item infos from the database."`
		RebuildTags    bool `long:"rebuild-tags" description:"Rescans all tag infos from the database."`
		Benchmark      bool `long:"benchmark" description:"Execute a 'slow' query a lot of times."`
		CheckpointFile string `long:"checkpoint-file" default:"/tmp/checkpoint.store" description:"Filename of the checkpoint file to read and write."`
		Postgres       string `long:"postgres" default:"postgres://postgres:password@localhost?sslmode=disable" description:"Connection-string for postgres database."`
		HttpListen     string `long:"http-listen" default:":8080" description:"Listen address for the rest api http server."`
	}

	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	storeLock := sync.RWMutex{}

	withReadLock := func(fn func()) {
		storeLock.RLock()
		defer storeLock.RUnlock()
		fn()
	}

	withWriteLock := func(fn func()) {
		storeLock.Lock()
		defer storeLock.Unlock()
		fn()
	}

	//var userIndex IterStore = BuildInvertedIndex("/tmp/users.csv", ExtractWordsSimple)
	//var invertedIndex IterStore = BuildInvertedIndex("/tmp/tags-2016-08-05.utf8.csv", ExtractWords)

	db := sqlx.MustConnect("postgres", opts.Postgres)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	storeState := StoreState{}
	store := &VarintStore{NewCppStore()}
	// store := &UncompressedStore{NewCppStore()}

	// read a checkpoint if there is one
	if _, err := os.Stat(opts.CheckpointFile); err == nil {
		log.Println("Found checkpoint at", opts.CheckpointFile)

		if err := ReadCheckpointFile(opts.CheckpointFile, &storeState, store); err != nil {
			log.Println("Reading checkpoint failed:", err)
		} else {
			log.Println("Checkpoint loaded, state:", storeState)
			log.Printf("Memory usage is now: %1.3fmb\n",
				float64(store.MemorySize()) / (1024.0 * 1024.0))
		}
	}

	// run garbage collection to cleanup all the stuff after setup
	runtime.GC()

	// do we maybe want to rebuild?
	if opts.RebuildItems {
		log.Println("Will re-read all items.")
		storeState.LastItemId = 0
	}

	if opts.RebuildTags {
		log.Println("Will re-read all tags.")
		storeState.LastTagId = 0
	}

	updateOnce := func() bool {
		log.Println("Looking for updates...")

		var currentStoreState StoreState
		withReadLock(func() {
			currentStoreState = storeState
		})

		updates, newState, more := FetchUpdates(db, storeState)
		log.Println("Number of updates to merge:", updates.KeyCount())

		start := time.Now()

		// now merge the updates into the store...
		withWriteLock(func() {
			MergeIterStores(store, updates)
			storeState = newState

			log.Println("Merging took", time.Since(start))
			log.Println("State is now:", storeState)
			log.Printf("Memory usage is now: %1.3fmb\n",
				float64(store.MemorySize()) / (1024.0 * 1024.0))
		})

		return more
	}

	updateJob := PreventConcurrency(func() {
		err := withRecovery(func() {
			for updateOnce() {
				// update again!
			}
		})

		if err != nil {
			log.Println("Error while updating:", err)
		}
	})

	writeCheckpoint := PreventConcurrency(func() {
		withReadLock(func() {
			start := time.Now()
			err := WriteCheckpointFile(opts.CheckpointFile, storeState, store)
			if err != nil {
				log.Println("Could not write checkpoint file:", err)
			}

			log.Println("Writing checkpoint took", time.Since(start))
		})
	})

	search := func(query string) (result []int32, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = r.(error)
			}
		}()

		parser := NewParser(strings.NewReader(query), func(str string) *ItemIterator {
			var hash uint32
			if str != "__all" {
				if len(str) < 2 || str[1] != ':' {
					str = CleanString(str)
				}

				hash = HashWord(str)
			}

			if store.Contains(hash) {
				return store.GetIterator(hash)
			} else {
				return NewSequenceIterator([]byte{})
			}
		})

		withReadLock(func() {
			iter := parser.Parse()
			result = IteratorToList(make([]int32, 0, 120), iter, 120)
		})

		return
	}

	if opts.Benchmark {
		fp, _ := os.Create("/tmp/profile.pprof")
		pprof.StartCPUProfile(fp)

		bar := pb.StartNew(3000)
		for i := 0; i < 3000; i++ {
			bar.Increment()
			search("((u:cha0s&f:sfw)-f:top)&webm")
		}

		pprof.StopCPUProfile()
		fp.Close()
		os.Exit(1)
	}

	// start updating in background
	go updateJob()

	cr := cron.New()
	cr.AddFunc("@every 6h", writeCheckpoint)
	cr.AddFunc("@every 15s", updateJob)
	cr.Start()

	startSearchServer(opts.HttpListen, search, writeCheckpoint)
}

func startSearchServer(httpListen string, search func(string) ([]int32, error), writeCheckpoint func()) {

	r := gin.Default()
	r.GET("/query/:query", func(c *gin.Context) {
		start := time.Now()
		items, err := search(c.ParamValue("query"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items": items,
			"duration": time.Since(start).String(),
		})
	})

	r.POST("/checkpoint", func(c*gin.Context) {
		start := time.Now()
		writeCheckpoint()
		c.JSON(http.StatusOK, gin.H{"duration": time.Since(start).String()})
	})

	r.Run(httpListen)
}

func withRecovery(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {

			err = fmt.Errorf("Cought an error in function %s: %s", fn, r)
		}
	}()

	fn()
	return nil
}

type ReverseByteReader struct {
	pos    int
	buffer []byte
}


