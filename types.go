package main

import "time"

// VvrBusStop represents all info from VVR belonging to one bus stop
type VvrBusStop struct {
	Cla    string `json:"cla"`
	ID     string `json:"id"`
	Label  string `json:"label"`
	Linien string `json:"linien"`
	Typ    string `json:"typ"`
	Value  string `json:"value"`
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
		RouteRef         string `json:"route_ref"`
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
	Linien   string
	City     string
	Elements []OsmElement
}

type MatchResult struct {
	ID              int
	VvrID           string
	Name            string
	IsIgnored       bool
	IsInOSM         bool
	IsInVVR         bool
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
	Rows            []MatchResult
	GenDate         time.Time
	IgnoredBusStops string
	Title           string
	Stats           Statistics
}
