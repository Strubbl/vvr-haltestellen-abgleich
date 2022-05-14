package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const cacheDir = "cache"
const cacheTimeOverpassInHours = 23
const cacheTimeVvrInHours = 167
const lockFile = ".lock"
const outputDir = "output"
const overpassDataFile = "overpass.json"
const templateName = "haltestellenabgleich"
const tmplDirectory = "tmpl"
const vvrDataFile = "vvr.json"
const vvrSearchURL = "https://vvr.verbindungssuche.de/fpl/suhast.php?&query="

// search area can be extended by append more areas with a pipe
const overpassSearchArea = "Vorpommern-Rügen"
const overpassURL = "http://overpass-api.de/api/interpreter?data="
const overpassQueryPrefix = "[out:json][timeout:600];area[boundary=administrative][admin_level=6][name~'("
const overpassQuerySuffix = ")']->.searchArea;(nw[\"public_transport\"=\"platform\"][\"bus\"](area.searchArea);node[\"public_transport\"=\"stop_position\"][\"bus\"](area.searchArea);node[\"highway\"=\"bus_stop\"](area.searchArea);rel[\"type\"=\"public_transport\"](area.searchArea););out;"

// tags
const tag_network = "Verkehrsgesellschaft Vorpommern-Rügen"
const tag_network_guid = "DE-MV-VVR"
const tag_network_short = "VVR"
const tag_operator = "Verkehrsgesellschaft Vorpommern-Rügen"

// warnings
const warning_network_tag_missing = "network tag is missing"
const warning_network_guid_tag_missing = "network:guid tag is missing"
const warning_network_short_tag_missing = "network:short tag is missing"
const warning_operator_tag_missing = "operator is missing"
const warning_network_tag_not_correct = "network tag is not correct"
const warning_network_guid_tag_not_correct = "network:guid tag is not correct"
const warning_network_short_tag_not_correct = "network:short tag is not correct"
const warning_operator_tag_not_correct = "operator is not correct"

// flags
var debug = flag.Bool("d", false, "get debug output (implies verbose mode)")
var verbose = flag.Bool("verbose", false, "verbose mode")

// non-const consts
var alphabet = [30]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "ä", "ö", "ü", "ß"}
var httpClient = &http.Client{Timeout: 1000 * time.Second}

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

// VvrData holds the VVR data and some meta data
type VvrData struct {
	CityResults []VvrCity
}

type OsmElement struct {
	Type string  `json:"type"`
	ID   int64   `json:"id"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Tags struct {
		Bench            string `json:"bench"`
		Bin              string `json:"bin"`
		Bus              string `json:"bus"`
		CheckDateShelter string `json:"check_date:shelter"`
		DeparturesBoard  string `json:"departures_board"`
		Highway          string `json:"highway"`
		Lit              string `json:"lit"`
		Name             string `json:"name"`
		Network          string `json:"network"`
		NetworkGuid      string `json:"network:guid"`
		NetworkShort     string `json:"network:short"`
		Operator         string `json:"operator"`
		PublicTransport  string `json:"public_transport"`
		Shelter          string `json:"shelter"`
		TactilePaving    string `json:"tactile_paving"`
		Wheelchair       string `json:"wheelchair"`
	} `json:"tags,omitempty"`
}

// OverpassData holds the OSM data queried via overpass api
type OverpassData struct {
	Version   float64 `json:"version"`
	Generator string  `json:"generator"`
	Osm3S     struct {
		TimestampOsmBase   time.Time `json:"timestamp_osm_base"`
		TimestampAreasBase time.Time `json:"timestamp_areas_base"`
		Copyright          string    `json:"copyright"`
	} `json:"osm3s"`
	Elements []OsmElement `json:"elements"`
}

// MatchedBusStops is a result of the merge of VVR data with OSM data
type MatchedBusStop struct {
	Name     string
	VvrID    string
	City     string
	Elements []OsmElement
}

type MatchResult struct {
	ID              int
	VvrID           string
	Name            string
	IsInVVR         bool
	IsInOSM         bool
	NrBusStops      int
	NrPlatforms     int
	NrStopPositions int
	OsmReference    string
}

type Statistics struct {
	VvrStops              int
	OsmStops              int
	OsmStopsNoName        int
	RemainingVvrStops     int
	RemainingOsmStops     int
	OsmStopsMatchingVvr   int
	VvrStopsWithOsmObject int
	WarningsSum           int
}

type TemplateData struct {
	Rows    []MatchResult
	GenDate time.Time
	Title   string
	Stats   Statistics
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
	case *OverpassData:
		if *debug {
			log.Println("readCurrentJSON: found *OverpassData type")
		}
		jsonFilePath = cacheDir + string(os.PathSeparator) + overpassDataFile

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
			log.Printf("file does not exist %s\n", jsonFilePath)
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
	case OverpassData:
		if *debug {
			log.Println("found OverpassData type")
		}
		if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
			os.Mkdir(cacheDir, os.ModePerm)
		}
		jsonFilePath = cacheDir + string(os.PathSeparator) + overpassDataFile
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

func getCityResultFromData(cityName string, vvr VvrData) *VvrCity {
	for i := 0; i < len(vvr.CityResults); i++ {
		if vvr.CityResults[i].SearchWord == cityName {
			return &vvr.CityResults[i]
		}
	}
	return nil
}

func doesOsmElementMatchVvrElement(osm OsmElement, vvrName string, cities []string) bool {
	search := [...]string{"-", "/", ",", "ä", "ö", "ü", "ß", "(", ")", ".", "hosanweg"}
	replace := [...]string{" ", " ", "", "ae", "oe", "ue", "ss", "", "", "", "hosangweg"}
	if len(search) != len(replace) {
		log.Panicln("search and replace arrays do not have the same length")
	}
	osmNameCleaned := strings.ToLower(osm.Tags.Name)
	vvrNameCleaned := strings.ToLower(vvrName)
	for i := 0; i < len(search); i++ {
		osmNameCleaned = strings.ReplaceAll(osmNameCleaned, search[i], replace[i])
		vvrNameCleaned = strings.ReplaceAll(vvrNameCleaned, search[i], replace[i])
	}
	// exact match
	if osmNameCleaned == vvrNameCleaned {
		return true
	}
	// prefix OSM name with a city
	for i := 0; i < len(cities); i++ {
		if strings.ToLower(cities[i])+" "+osmNameCleaned == vvrNameCleaned {
			return true
		}
	}
	return false
}

func writeTemplateToHTML(templateData TemplateData) {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		os.Mkdir(outputDir, os.ModePerm)
	}
	f, err := os.Create(outputDir + string(os.PathSeparator) + templateName + ".html")
	if err != nil {
		log.Println("writeTemplateToHTML", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Println("writeTemplateToHTML", err)
		}
	}()

	htmlSource, err := template.New(templateName + ".tmpl").ParseFiles(tmplDirectory + "/" + templateName + ".tmpl")
	if err != nil {
		log.Println("writeTemplateToHTML", err)
	}
	htmlSource.Execute(f, templateData)
}

func doesNameExistAlreadyInArray(mbs []MatchedBusStop, name string) int {
	for i := 0; i < len(mbs); i++ {
		if mbs[i].Name == name {
			return i
		}
	}
	return -1
}

// main
func main() {
	start := time.Now()
	defer printElapsedTime(start)

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

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

	// check if lock file exists and exit, so we do not run this process two times
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		if *debug {
			log.Printf("no lockfile %s present\n", lockFile)
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

	// get the VVR data
	lenSearchWords := len(alphabet) * len(alphabet)
	searchWords := make([]string, lenSearchWords)
	for i := 0; i < len(alphabet); i++ {
		for k := 0; k < len(alphabet); k++ {
			searchWords[i*len(alphabet)+k] = alphabet[i] + alphabet[k]
		}
	}
	var newVvr VvrData
	for i := 0; i < len(searchWords); i++ {
		var newVvrCity VvrCity
		var newResult []VvrBusStop
		isNewApiCallNeeded := false
		oldVvrCity := getCityResultFromData(searchWords[i], oldVvr)
		if oldVvrCity != nil {
			if *debug {
				log.Printf("found old result for %s, checking for timestamp %s\n", oldVvrCity.SearchWord, oldVvrCity.ResultTimeStamp)
			}
			cacheTime := time.Now().Add(-1 * cacheTimeVvrInHours * time.Hour)
			if oldVvrCity.ResultTimeStamp.Before(cacheTime) {
				if *debug {
					log.Printf("data in cache is older than %d hours, trying to get fresh data\n", cacheTimeVvrInHours)
				}
				isNewApiCallNeeded = true
			}
		} else {
			isNewApiCallNeeded = true
		}
		if isNewApiCallNeeded {
			getURL := vvrSearchURL + url.QueryEscape(searchWords[i])
			err = getJson(getURL, &newResult)
			if err != nil {
				log.Println("error getting http json for", getURL)
				log.Println("error is", err)
				if oldVvrCity != nil {
					log.Printf("reusing old cache data for %s due to the GET error\n", searchWords[i])
					newVvr.CityResults = append(newVvr.CityResults, *oldVvrCity)
				}
				continue
			}
			newVvrCity.SearchWord = searchWords[i]
			newVvrCity.ResultTimeStamp = time.Now()
			newVvrCity.Result = newResult
			newVvr.CityResults = append(newVvr.CityResults, newVvrCity)
		} else {
			newVvr.CityResults = append(newVvr.CityResults, *oldVvrCity)
		}
	}
	err = writeNewJSON(newVvr)
	if err != nil {
		log.Printf("error writing json with VVR data: %v\n", err)
	}
	// TODO get all city names from newVvr
	var extractedCities []string
	for i := 0; i < len(newVvr.CityResults); i++ {
		for k := 0; k < len(newVvr.CityResults[i].Result); k++ {
			bussi := newVvr.CityResults[i].Result[k].Value
			if strings.Contains(bussi, ",") {
				bussiCity := strings.Split(bussi, ",")[0]
				isCityKnown := false
				for n := 0; n < len(extractedCities); n++ {
					if extractedCities[n] == bussiCity {
						isCityKnown = true
						break
					}
				}
				if !isCityKnown {
					extractedCities = append(extractedCities, bussiCity)
				}
			}
		}
	}
	if *debug {
		log.Println("extractedCities:", extractedCities, len(extractedCities))
	}
	// get OSM data
	overpassQuery := overpassURL + overpassQueryPrefix + overpassSearchArea + overpassQuerySuffix
	if *debug {
		log.Println("overpassQuery:", overpassQuery)
	}
	var oldOverpassData OverpassData
	err = readCurrentJSON(&oldOverpassData)
	if err != nil {
		removeLockFile(lockFile)
		panic(err)
	}
	var newOverpassData OverpassData
	cacheTime := time.Now().Add(-1 * cacheTimeOverpassInHours * time.Hour)
	isWriteOverpassJson := false
	if oldOverpassData.Osm3S.TimestampOsmBase.Before(cacheTime) {
		err = getJson(overpassQuery, &newOverpassData)
		if err != nil {
			log.Println("error getting http json for", overpassQuery)
			log.Println("error is", err)
			log.Println("reusing old overpass cache data due to the GET error")
			newOverpassData = oldOverpassData
		}
		isWriteOverpassJson = true
	} else {
		if *debug {
			log.Println("reusing old overpass data from cache, data is not older than hours:", cacheTimeOverpassInHours)
		}
		newOverpassData = oldOverpassData
	}
	if isWriteOverpassJson {
		err = writeNewJSON(newOverpassData)
		if err != nil {
			log.Printf("error writing json with VVR data: %v\n", err)
		}
	}

	totalOsmElements := len(newOverpassData.Elements)
	if *debug {
		log.Println("newOverpassData.Elements before matching:", len(newOverpassData.Elements))
	}
	// match VVR data with OSM Elements
	var mbs []MatchedBusStop
	insaneLoops := 0
	for i := 0; i < len(newVvr.CityResults); i++ {
		for k := 0; k < len(newVvr.CityResults[i].Result); k++ {
			oneBusStop := newVvr.CityResults[i].Result[k]
			var oneMatch MatchedBusStop
			oneMatch.Name = oneBusStop.Value
			oneMatch.VvrID = oneBusStop.ID
			// use VVR ID to remove duplicate VVR entities
			vvrIsDuplicate := false
			for p := 0; p < len(mbs); p++ {
				if mbs[p].VvrID == oneMatch.VvrID {
					vvrIsDuplicate = true
				}
			}
			if !vvrIsDuplicate {
				oneMatch.City = newVvr.CityResults[i].SearchWord
				var removeOsmElementsByID []int64
				for m := 0; m < len(newOverpassData.Elements); m++ {
					if doesOsmElementMatchVvrElement(newOverpassData.Elements[m], oneMatch.Name, extractedCities) {
						oneMatch.Elements = append(oneMatch.Elements, newOverpassData.Elements[m])
						removeOsmElementsByID = append(removeOsmElementsByID, newOverpassData.Elements[m].ID)
					}
					insaneLoops++
				}
				// remove the elements we already matched
				for n := 0; n < len(removeOsmElementsByID); n++ {
					toDeleteOsmId := removeOsmElementsByID[n]
					for p := 0; p < len(newOverpassData.Elements); p++ {
						if newOverpassData.Elements[p].ID == toDeleteOsmId {
							newOverpassData.Elements = append(newOverpassData.Elements[:p], newOverpassData.Elements[p+1:]...)
							continue
						}
					}
				}
				mbs = append(mbs, oneMatch)
			}
		}
	}
	remainingOsmElements := len(newOverpassData.Elements)
	if *debug {
		log.Println("insane looping finished:", insaneLoops)
		log.Println("newOverpassData.Elements left after matching:", len(newOverpassData.Elements))
	}

	// append remaining OSM elements, which couldn't be matched
	for i := 0; i < len(newOverpassData.Elements); i++ {
		index := doesNameExistAlreadyInArray(mbs, newOverpassData.Elements[i].Tags.Name)
		if index >= 0 {
			mbs[index].Elements = append(mbs[index].Elements, newOverpassData.Elements[i])
		} else {
			var notInVvrButInOsm MatchedBusStop
			notInVvrButInOsm.Name = newOverpassData.Elements[i].Tags.Name
			notInVvrButInOsm.Elements = append(notInVvrButInOsm.Elements, newOverpassData.Elements[i])
			mbs = append(mbs, notInVvrButInOsm)
		}
	}

	vvrBusStopSum := 0
	remainingVvrStops := 0
	osmStopsNoName := 0
	warningsSum := 0
	result := make([]MatchResult, len(mbs))
	for i := 0; i < len(mbs); i++ {
		result[i].ID = i + 1
		result[i].VvrID = mbs[i].VvrID
		result[i].IsInOSM = false
		if len(mbs[i].Elements) > 0 {
			result[i].IsInOSM = true
		}
		result[i].IsInVVR = false
		if mbs[i].VvrID != "" {
			result[i].IsInVVR = true
			vvrBusStopSum++
		}
		result[i].NrPlatforms = 0
		result[i].NrStopPositions = 0
		result[i].Name = mbs[i].Name
		result[i].OsmReference = ""
		for k := 0; k < len(mbs[i].Elements); k++ {
			object := mbs[i].Elements[k]
			object_id := strconv.FormatInt(object.ID, 10)
			objectURL := "http://osm.org/" + object.Type + "/" + object_id
			// OSM Reference column filling Start
			josm_link := "<a href=\"http://127.0.0.1:8111/load_object?new_layer=false&objects=" + string(object.Type[0]) + object_id + "\" target=\"hiddenIframe\" title=\"edit in JOSM\">(j)</a>"
			result[i].OsmReference = result[i].OsmReference + "<p><a href=\"" + objectURL + "\">" + object.Type + " " + object_id + "</a> " + josm_link
			if object.Type != "relation" && (object.Tags.PublicTransport != "stop_position" || object.Tags.Highway == "bus_stop") {
				if object.Tags.Network == "" {
					result[i].OsmReference = result[i].OsmReference + "<br />- " + warning_network_tag_missing
					warningsSum++
				} else if object.Tags.Network != tag_network {
					result[i].OsmReference = result[i].OsmReference + "<br />- " + warning_network_tag_not_correct + ". " + object.Tags.Network + " instead of network=" + tag_network
					warningsSum++
				}
				if object.Tags.NetworkGuid == "" {
					result[i].OsmReference = result[i].OsmReference + "<br />- " + warning_network_guid_tag_missing
					warningsSum++
				} else if object.Tags.NetworkGuid != tag_network_guid {
					result[i].OsmReference = result[i].OsmReference + "<br />- " + warning_network_guid_tag_not_correct + ". " + object.Tags.NetworkGuid + " instead of network:guid=" + tag_network_guid
					warningsSum++
				}
				if object.Tags.NetworkShort == "" {
					result[i].OsmReference = result[i].OsmReference + "<br />- " + warning_network_short_tag_missing
					warningsSum++
				} else if object.Tags.NetworkShort != tag_network_short {
					result[i].OsmReference = result[i].OsmReference + "<br />- " + warning_network_short_tag_not_correct + ". " + object.Tags.NetworkShort + " instead of network:short" + tag_network_short
					warningsSum++
				}
			}
			if object.Tags.Operator == "" {
				result[i].OsmReference = result[i].OsmReference + "<br />- " + warning_operator_tag_missing
				warningsSum++
			} else if object.Tags.Operator != tag_operator {
				result[i].OsmReference = result[i].OsmReference + "<br />- " + warning_operator_tag_not_correct + ". " + object.Tags.Operator + " instead of operator=" + tag_operator
				warningsSum++
			}
			result[i].OsmReference = result[i].OsmReference + "</p>"
			// OSM Reference column filling End
			if object.Tags.Highway == "bus_stop" {
				result[i].NrBusStops++
			}
			if object.Tags.PublicTransport == "stop_position" {
				result[i].NrStopPositions++
			}
			if object.Tags.PublicTransport == "platform" {
				result[i].NrPlatforms++
			}
			if object.Tags.Name == "" {
				osmStopsNoName++
			}
		}
		if result[i].IsInVVR && len(mbs[i].Elements) == 0 {
			remainingVvrStops++
		}
	}

	var templateData TemplateData
	templateData.Rows = result
	templateData.GenDate = time.Now()
	templateData.Title = "VVR-OSM Haltestellenabgleich"
	templateData.Stats.VvrStops = vvrBusStopSum
	templateData.Stats.OsmStops = totalOsmElements
	templateData.Stats.OsmStopsNoName = osmStopsNoName
	templateData.Stats.RemainingVvrStops = remainingVvrStops
	templateData.Stats.RemainingOsmStops = remainingOsmElements
	templateData.Stats.OsmStopsMatchingVvr = totalOsmElements - remainingOsmElements
	templateData.Stats.VvrStopsWithOsmObject = vvrBusStopSum - remainingVvrStops
	templateData.Stats.WarningsSum = warningsSum
	writeTemplateToHTML(templateData)
}
