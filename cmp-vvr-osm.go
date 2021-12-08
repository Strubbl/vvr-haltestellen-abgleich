package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

const lockFile = ".lock"
const cacheDir = "cache"
const vvrDataFile = "vvr.json"

// flags
var debug = flag.Bool("d", false, "get debug output (implies verbose mode)")
var verbose = flag.Bool("verbose", false, "verbose mode")

// non-const consts
var cities = [...]string{"Altefähr", "Kramerhof", "Parow", "Prohn", "Stralsund"}

func printElapsedTime(start time.Time) {
	if *debug {
		log.Printf("printElapsedTime: time elapsed %.2fs\n", time.Since(start).Seconds())
	}
}

// type definitions
// VvrBusStop represents all info from VVR belonging to one bus stop
type VvrBusStop struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
	Label string `json:"label"`
}

// VvrData holds the json response from the VVR search api
type VvrData struct {
	VvrBusStops []VvrBusStop
}

// CompareDataSet holds the VVR data, the OSM data and some meta data for one city
type CompareDataSet struct {
	SearchWord string
	Vvr        VvrData
}

// functions
func removeLockFile(lf string) {
	if *debug {
		log.Printf("removeLockFile: trying to delete %s\n", lf)
	}
	err := os.Remove(lf)
	if err != nil {
		log.Printf("removeLockFile: error while removing lock file %s\n", lf)
		log.Panic(err)
	}
}

func main() {
	start := time.Now()
	defer printElapsedTime(start)

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Flag handling
	flag.Parse()
	if *debug && len(flag.Args()) > 0 {
		log.Printf("non-flag args=%v", strings.Join(flag.Args(), " "))
	}

	if *verbose && !*debug {
		log.Printf("verbose mode")
	}
	if *debug {
		log.Printf("debug mode")
		// debug implies verbose
		*verbose = true
	}

	// check if lock file exists and exit, so we do not run this process two times
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		if *debug {
			log.Printf("no lockfile %s present", lockFile)
		}
	} else {
		fmt.Printf("abort: lock file exists %s\n", lockFile)
		os.Exit(1)
	}

	// check if lock file exists and exit, so we do not run this process two times
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		if *debug {
			log.Printf("main: no lockfile %s present", lockFile)
		}
	} else {
		fmt.Printf("abort: lock file exists %s\n", lockFile)
		os.Exit(1)
	}
	// create lock file and delete it on exit of main
	err := ioutil.WriteFile(lockFile, nil, 0644)
	if err != nil {
		if *debug {
			log.Println("main: error while writing lock file")
		}
		panic(err)
	}

	defer removeLockFile(lockFile)
}
