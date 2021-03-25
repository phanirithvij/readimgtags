package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ignore "github.com/sabhiram/go-gitignore"
)

var gitignore *ignore.GitIgnore
var err error

// silly metrics
var (
	subDirCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "exif_processed_subdir_count",
			Help: "The number of subdirectories processed till now",
		})

	ignoredDirCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "exif_ignored_subdir_count",
			Help: "The number of subdirectories ignored till now",
		})

	failedFilesCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "exif_failed_file_count",
			Help: "The number of files which we failed to process till now",
		})

	filesSizeCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "exif_processed_files_size",
			Help: "The total size of files processed till now",
		})
)

func init() {
	prometheus.MustRegister(prometheus.NewBuildInfoCollector())
	prometheus.MustRegister(filesSizeCounter)
	prometheus.MustRegister(subDirCounter)
	prometheus.MustRegister(ignoredDirCounter)
	prometheus.MustRegister(failedFilesCounter)

	gitignore, err = ignore.CompileIgnoreFile(".gitignore")
	if err != nil {
		panic(err)
	}
}

func shoudIgnore(path string) (bool, *ignore.IgnorePattern) {
	base := filepath.Base(path)
	if base[0] == '.' || base[0] == '_' {
		// hidden files/folders
		return true, nil
	}
	return gitignore.MatchesPathHow(path)
}

var wg sync.WaitGroup

func walkDir(dir string) {
	defer subDirCounter.Add(1)
	defer wg.Done()
	visit := func(path string, f os.FileInfo, err error) error {
		if f.IsDir() && path != dir {
			// remove useless dirs
			ignr, reason := shoudIgnore(path)
			if ignr {
				if reason != nil {
					ignoredDirCounter.Add(1)
					fmt.Printf("Ignored: %s; Reason: Occurs at: line %d %s\n", path, reason.LineNo, reason.Line)
				}
				// else hidden file/folders
				return filepath.SkipDir
			}
			wg.Add(1)
			go walkDir(path)
			return filepath.SkipDir
		}
		if ignr, _ := shoudIgnore(path); ignr {
			return nil
		}
		if f.Mode().IsRegular() {
			if validImg, err := printExif(path); validImg {
				// only if a valid image
				if err != nil {
					failedFilesCounter.Add(1)
				} else {
					filesSizeCounter.Add(float64(f.Size()))
				}
			}
		}
		return nil
	}

	filepath.Walk(dir, visit)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fname := os.Args[1]
	f, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// https://github.com/prometheus/client_golang/blob/master/examples/random/main.go
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))

	go func() {
		log.Fatal(http.ListenAndServe("localhost:8010", nil))
	}()

	wg.Add(1)
	walkDir(fname)
	wg.Wait()
	fmt.Println("Done....")
}
