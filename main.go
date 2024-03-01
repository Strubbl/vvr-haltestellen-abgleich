package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// main
func main() {
	start := time.Now()
	defer printElapsedTime(start)
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	handleFlags()

	// check if lock file exists and exit, so we do not run this process two times
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		if *verbose {
			log.Printf("no lockfile %s present\n", lockFile)
		}
	} else {
		fmt.Printf("abort: lock file exists %s\n", lockFile)
		os.Exit(1)
	}

	// create lock file and delete it on exit of main
	err := os.WriteFile(lockFile, nil, 0644)
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
	if *verbose {
		log.Println("extractedCities:", extractedCities, len(extractedCities))
	}
	// get OSM data
	overpassQuery := overpassURL + overpassQueryPrefix + overpassSearchArea + overpassQuerySuffix
	if *verbose {
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
		if *verbose {
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
	if *verbose {
		log.Println("matching VVR data with OSM Elements")
	}
	var mbs []MatchedBusStop
	insaneLoops := 0
	for i := 0; i < len(newVvr.CityResults); i++ {
		for k := 0; k < len(newVvr.CityResults[i].Result); k++ {
			oneBusStop := newVvr.CityResults[i].Result[k]
			var oneMatch MatchedBusStop
			oneMatch.Name = oneBusStop.Value
			oneMatch.Linien = oneBusStop.Linien
			oneMatch.VvrID = oneBusStop.ID
			// use VVR ID to remove duplicate VVR entities
			vvrIsDuplicate := false
			for p := 0; p < len(mbs); p++ {
				if mbs[p].VvrID == oneMatch.VvrID {
					vvrIsDuplicate = true
				}
			}
			vvrIsSpecialDestination := false
			for specDest := 0; specDest < len(ignoreVvrStops); specDest++ {
				if strings.Contains(oneMatch.Name, ignoreVvrStops[specDest]) {
					vvrIsSpecialDestination = true
				}
			}
			if !vvrIsDuplicate && !vvrIsSpecialDestination {
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
	if *verbose {
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
		result[i].IsIgnored = false
		result[i].OsmReference = ""
		for k := 0; k < len(mbs[i].Elements); k++ {
			object := mbs[i].Elements[k]
			object_id := strconv.FormatInt(object.ID, 10)
			objectURL := "http://osm.org/" + object.Type + "/" + object_id
			// OSM Reference column filling Start
			josm_link := "<a href=\"http://127.0.0.1:8111/load_object?new_layer=false&objects=" + string(object.Type[0]) + object_id + "\" target=\"hiddenIframe\" title=\"edit in JOSM\">(j)</a>"
			result[i].OsmReference = result[i].OsmReference + "<p><a href=\"" + objectURL + "\">" + object.Type + " " + object_id + "</a> " + josm_link
			// ignore certain bus stops having a known operator
			value, exists := ignoreBusStopsWithOperators[object.Tags.Operator]
			if exists {
				value++
				ignoreBusStopsWithOperators[object.Tags.Operator] = value
				if *debug {
					log.Println("operator", object.Tags.Operator, "shall be ignored for object", objectURL)
				}
				result[i].OsmReference = result[i].OsmReference + " (Operator is " + object.Tags.Operator + ")</p>"
				result[i].IsIgnored = true
				// skip further processing for this bus stop because it is not VVR but a different operator
				continue
			}
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
			// check route_ref for platforms only
			targetRouteRef, err := convertLinienToRouteRef(mbs[i].Linien)
			if err != nil {
				log.Println("convertLinienToRouteRef failed with error:", err)
			}
			if object.Tags.PublicTransport == "platform" && object.Tags.RouteRef == "" && targetRouteRef != "" {
				result[i].OsmReference = result[i].OsmReference + "<br />- route_ref missing:<br><code>route_ref=" + targetRouteRef + "</code>"
				warningsSum++
			}
			if object.Tags.PublicTransport == "platform" && object.Tags.RouteRef != "" && targetRouteRef != "" && object.Tags.RouteRef != targetRouteRef {
				result[i].OsmReference = result[i].OsmReference + "<br />- existing <code>route_ref=" + object.Tags.RouteRef + "</code> does not match calculated <code>route_ref=" + targetRouteRef + "</code>"
				warningsSum++
			}
			// check operator
			if object.Tags.Operator == "" {
				result[i].OsmReference = result[i].OsmReference + "<br />- " + warning_operator_tag_missing + warning_operator_might_be_vvr
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
	templateData.IgnoredBusStops = fmt.Sprint(ignoreBusStopsWithOperators)
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
