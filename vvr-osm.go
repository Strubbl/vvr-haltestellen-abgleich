package main

import (
	"log"
	"regexp"
	"sort"
	"strconv"
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

func convertLinienToRouteRef(linien string) (string, error) {
	routeRef := ""
	if linien == "" {
		return routeRef, nil
	}
	var linesArr []int
	var err error
	var re = regexp.MustCompile(`<span.*?>([0-9]+)</span.*?>`)
	res := re.FindAllStringSubmatch(linien, -1)
	log.Println("linien:", linien)
	for i := range res {
		log.Println("res[i]:", res[i])
		line, err := strconv.Atoi(res[i][1])
		if err != nil {
			return "", err
		}
		linesArr = append(linesArr, line)
	}
	sort.Ints(linesArr)
	for i := 0; i < len(linesArr); i++ {
		routeRef += strconv.Itoa(linesArr[i])
		if i < len(linesArr)-1 {
			routeRef += ";"
		}
	}
	return routeRef, err
}
