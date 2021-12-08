package main

import (
	"encoding/json"
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

// VvrSearchResult holds the json response from the VVR search api
type VvrSearchResult struct {
	VvrBusStops []VvrBusStop
}

// VvrCity holds the VVR data and what was searched for via the API
type VvrCity struct {
	SearchWord      string
	ResultTimeStamp time.Time
	Vvr             VvrSearchResult
}

// VvrData holds the VVR data, the OSM data and some meta data for one city
type VvrData struct {
	cityResults []VvrCity
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

func readCurrentJSON(i interface{}) error {
	if *debug {
		log.Println("readCurrentJSON")
	}
	var jsonFilePath string
	if *debug {
		log.Println("readCurrentJSON: given type:")
		log.Printf("%T\n", i)
	}
	switch i.(type) {
	case *VvrData:
		if *debug {
			log.Println("readCurrentJSON: found *VvrData type")
		}
		jsonFilePath = string(cacheDir) + string(os.PathSeparator) + vvrDataFile
	default:
		log.Fatalln("readCurrentJSON: unkown type for reading json")
		return nil
	}

	if *debug {
		log.Println("readCurrentJSON: jsonFilePath is", jsonFilePath)
	}
	if _, err := os.Stat(jsonFilePath); os.IsNotExist(err) {
		// in case file does not exist, we cannot prefill the data from json
		if *verbose { // not fatal, just start with a new one
			log.Printf("file does not exist %s", jsonFilePath)
		}
		return nil
	}
	b, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		if *debug {
			log.Println("readCurrentJSON: error while ioutil.ReadFile", err)
		}
		fmt.Println(err)
		return err
	}
	err = json.Unmarshal(b, i)
	if err != nil {
		if *debug {
			log.Println("readCurrentJSON: error while json.Unmarshal", err)
		}
		return err
	}
	return nil
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

	if *verbose {
		log.Println("reading data json file into memory")
	}
	var vvr VvrData
	err = readCurrentJSON(&vvr)
	if err != nil {
		removeLockFile(lockFile)
		panic(err)
	}
	log.Println(vvr)

}
