package main

import (
	"log"
	"strings"
)

func getCityResultFromData(cityName string, vvr VvrData) *VvrCity {
	for i := 0; i < len(vvr.CityResults); i++ {
		if vvr.CityResults[i].SearchWord == cityName {
			return &vvr.CityResults[i]
		}
	}
	return nil
}

func doesOsmElementMatchVvrElement(osm OsmElement, vvrName string, cities []string) bool {
	if len(searchStopName) != len(replaceStopName) {
		log.Panicln("search and replace arrays do not have the same length")
	}
	osmNameCleaned := strings.ToLower(osm.Tags.Name)
	vvrNameCleaned := strings.ToLower(vvrName)
	for i := 0; i < len(searchStopName); i++ {
		osmNameCleaned = strings.ReplaceAll(osmNameCleaned, searchStopName[i], replaceStopName[i])
		vvrNameCleaned = strings.ReplaceAll(vvrNameCleaned, searchStopName[i], replaceStopName[i])
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

func doesNameExistAlreadyInArray(mbs []MatchedBusStop, name string) int {
	for i := 0; i < len(mbs); i++ {
		if mbs[i].Name == name {
			return i
		}
	}
	return -1
}
