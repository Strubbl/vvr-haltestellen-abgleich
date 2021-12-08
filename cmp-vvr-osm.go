package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const lockFile = ".lock"
const cacheDir = "cache"
const vvrDataFile = "vvr.json"
const vvrSearchURL = "https://vvr.verbindungssuche.de/fpl/suhast.php?&query="

// flags
var debug = flag.Bool("d", false, "get debug output (implies verbose mode)")
var verbose = flag.Bool("verbose", false, "verbose mode")

// non-const consts
var cities = [...]string{"Altefähr", "Kramerhof", "Parow", "Prohn", "Stralsund"}
var httpClient = &http.Client{Timeout: 10 * time.Second}

// type definitions
// VvrBusStop represents all info from VVR belonging to one bus stop
type VvrBusStop struct {
	ID    string `json:"id"`
	Value string `json:"value"`
	Label string `json:"label"`
}

// VvrCity holds the VVR data and some meta data about it
type VvrCity struct {
	SearchWord      string
	ResultTimeStamp time.Time
	Result          []VvrBusStop
}

// VvrData holds the VVR data, the OSM data and some meta data for one city
type VvrData struct {
	CityResults []VvrCity
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
		jsonFilePath = cacheDir + string(os.PathSeparator) + vvrDataFile
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

func writeNewJSON(i interface{}) error {
	if *debug {
		log.Println("writeNewJSON: given type:")
		log.Printf("%T\n", i)
	}
	var jsonFilePath string
	switch i.(type) {
	case VvrData:
		if *debug {
			log.Println("found VvrData type")
		}
		if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
			os.Mkdir(cacheDir, os.ModePerm)
		}
		jsonFilePath = cacheDir + string(os.PathSeparator) + vvrDataFile
	default:
		return errors.New("unkown data type for writing json")
	}
	b, err := json.Marshal(i)
	if err != nil {
		if *debug {
			log.Println("writeNewJSON: error while marshalling data json", err)
		}
		return err
	}
	err = ioutil.WriteFile(jsonFilePath, b, 0644)
	if err != nil {
		if *debug {
			log.Println("writeNewJSON: error while writing data json", err)
		}
		return err
	}
	return nil
}

func getJson(url string, target interface{}) error {
	r, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func printElapsedTime(start time.Time) {
	if *debug {
		log.Printf("printElapsedTime: time elapsed %.2fs\n", time.Since(start).Seconds())
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

	if *verbose {
		log.Println("reading data json file into memory")
	}
	var oldVvr VvrData
	err = readCurrentJSON(&oldVvr)
	if err != nil {
		removeLockFile(lockFile)
		panic(err)
	}

	var newVvr VvrData
	for i := 0; i < len(cities); i++ {
		var newVvrCity VvrCity
		var newResult []VvrBusStop
		getURL := vvrSearchURL + cities[i]
		err = getJson(getURL, &newResult)
		if err != nil {
			log.Println("error getting http json for", getURL)
			log.Println("error is", err)
			continue
		}
		newVvrCity.SearchWord = cities[i]
		newVvrCity.ResultTimeStamp = time.Now()
		newVvrCity.Result = newResult
		newVvr.CityResults = append(newVvr.CityResults, newVvrCity)
	}
	log.Println(newVvr)
	err = writeNewJSON(newVvr)
	if err != nil {
		log.Printf("error writing json: %v", err)
	}

}
