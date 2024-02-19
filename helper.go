package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"
)

func removeLockFile(lf string) {
	if *verbose {
		log.Printf("removeLockFile: trying to delete %s\n", lf)
	}
	err := os.Remove(lf)
	if err != nil {
		log.Printf("removeLockFile: error while removing lock file %s\n", lf)
		log.Panic(err)
	}
}

func printElapsedTime(start time.Time) {
	if *verbose {
		log.Printf("printElapsedTime: time elapsed %.2fs\n", time.Since(start).Seconds())
	}
}

func handleFlags() {
	// Flag handling
	flag.Parse()
	if *debug && len(flag.Args()) > 0 {
		log.Printf("non-flag args=%v\n", strings.Join(flag.Args(), " "))
	}

	if *verbose && !*debug {
		log.Println("verbose mode")
	}
	if *debug {
		log.Println("debug mode")
		// debug implies verbose
		*verbose = true
	}

}
