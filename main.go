package main

import (
	"fmt"
	"hash/fnv"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/cznic/sortutil"
	"github.com/jmoiron/sqlx"
	"gopkg.in/cheggaaa/pb.v1"

	"runtime/pprof"
	"sync/atomic"

	"github.com/jessevdk/go-flags"
	_ "github.com/lib/pq"
	"github.com/robfig/cron"
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

func preventConcurrency(fn func()) func() {
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
		RebuildItems   bool   `long:"rebuild-items" description:"Rescans all item infos from the database."`
		RebuildTags    bool   `long:"rebuild-tags" description:"Rescans all tag infos from the database."`
		Benchmark      bool   `long:"benchmark" description:"Execute a 'slow' query a lot of times."`
		CheckpointFile string `long:"checkpoint-file" default:"/tmp/checkpoint.store" description:"Filename of the checkpoint file to read and write."`
		Postgres       string `long:"postgres" default:"postgres://postgres:password@localhost?sslmode=disable" description:"Connection-string for postgres database."`
		HttpListen     string `long:"http-listen" default:":8080" description:"Listen address for the rest api http server."`
	}

	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	db := sqlx.MustConnect("postgres", opts.Postgres)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	storeState := StoreState{}
	store := NewIterStore(NewCppStore())

	// read a checkpoint if there is one
	if st, err := os.Stat(opts.CheckpointFile); err == nil && st.Size() > 0 {
		log.WithField("file", opts.CheckpointFile).Info("Found checkpoint to load")

		if err := ReadCheckpointFile(opts.CheckpointFile, &storeState, store); err != nil {
			log.WithError(err).Warn("Reading checkpoint failed")
		} else {
			log.WithField("state", storeState).
				WithField("memoryUsage", store.MemorySize()).
				Info("Checkpoint loaded, state:")
		}
	}

	// run garbage collection to cleanup all the stuff after setup
	log.Debug("Running garbage collection now.")
	runtime.GC()

	// do we maybe want to rebuild?
	if opts.RebuildItems {
		log.Info("Will re-read all items.")
		storeState.LastItemId = 0
	}

	if opts.RebuildTags {
		log.Info("Will re-read all tags.")
		storeState.LastTagId = 0
	}

	actions := &storeActions{
		store:      store,
		storeState: storeState,
	}

	if opts.Benchmark {
		log.Info("Running benchmarks.")
		start := time.Now()
		RunBenchmarks(actions)

		log.Infof("Benchmarking took %s, exiting now.", time.Since(start))
		os.Exit(1)
	}

	updateJob := preventConcurrency(func() {
		err := withRecovery("update", func() {
			for actions.UpdateOnce(db) {
				// update again!
			}
		})

		if err != nil {
			log.Println("Error while updating:", err)
		}
	})

	// start updating in background now to get the most recent state.
	go updateJob()

	cr := cron.New()

	cr.AddFunc("@every 6h", preventConcurrency(func() {
		actions.WriteCheckpoint(opts.CheckpointFile)
	}))

	cr.AddFunc("@every 15s", updateJob)
	cr.Start()

	restApi(opts.HttpListen, actions, opts.CheckpointFile)
}

func RunBenchmarks(actions *storeActions) {
	fp, _ := os.Create("/tmp/profile.pprof")
	pprof.StartCPUProfile(fp)

	bar := pb.StartNew(3000)
	for i := 0; i < 30; i++ {
		for j := 0; j < 100; j++ {
			// this query produces only 3 hits, but we need to search nearly all posts.
			actions.Search("((u:cha0s&f:sfw)-f:top)&webm")
		}

		bar.Add(100)
	}

	pprof.StopCPUProfile()
	fp.Close()
}

func withRecovery(name string, fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {

			err = fmt.Errorf("Cought an error in function '%s': %s", name, r)
		}
	}()

	fn()
	return nil
}
