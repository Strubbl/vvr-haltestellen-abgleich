package main

import (
	"flag"
	"net/http"
	"time"
)

const cacheDir = "cache"
const cacheTimeOverpassInHours = 8
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
const overpassQuerySuffix = ")']->.searchArea;(nw[\"public_transport\"=\"platform\"][\"bus\"](area.searchArea);node[\"public_transport\"=\"stop_position\"][\"bus\"](area.searchArea);node[\"highway\"=\"bus_stop\"](area.searchArea);rel[\"type\"=\"public_transport\"][\"bus\"](area.searchArea););out;"

// tags
const tag_network = "Verkehrsgesellschaft Vorpommern-Rügen"
const tag_network_guid = "DE-MV-VVR"
const tag_network_short = "VVR"
const tag_operator = "Verkehrsgesellschaft Vorpommern-Rügen"

//const uns_luett_bahn_tag_network = "Uns lütt Bahn"
//const uns_luett_bahn_tag_operator = "Rügen-Bahnen"

// warnings
const warning_network_tag_missing = "network tag is missing"
const warning_network_guid_tag_missing = "network:guid tag is missing"
const warning_network_short_tag_missing = "network:short tag is missing"
const warning_operator_tag_missing = "operator is missing"
const warning_operator_might_be_vvr = ", might be operator=" + tag_operator
const warning_network_tag_not_correct = "network tag is not correct"
const warning_network_guid_tag_not_correct = "network:guid tag is not correct"
const warning_network_short_tag_not_correct = "network:short tag is not correct"
const warning_operator_tag_not_correct = "operator is not correct"

// flags
var debug = flag.Bool("d", false, "get debug output (implies verbose mode)")
var verbose = flag.Bool("verbose", false, "verbose mode")

// non-const consts
var alphabet = [30]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "ä", "ö", "ü", "ß"}
var ignoreBusStopsWithOperators = map[string]int{
	"Darßbahn":                 0,
	"Rügen-Bahnen":             0,
	"Kap-Arkona-Bahn":          0,
	"Busunternehmen M. Scholz": 0,
	"Deutsche Bahn":            0,
	"Flixbus":                  0,
	"Gemeinde Binz":            0,
	"Regenbogencamp Nonnevitz": 0,
}
var httpClient = &http.Client{Timeout: 1000 * time.Second}
var ignoreVvrStops = []string{"SEV", "(Workshop)", "Schulbus", "Wagen defekt", "Stralsund, Velgast", "Stralsund, Tribseer Wiesen", "Sonderfahrt", "Probefahrt", "Stralsund, O.-Palme-Platz Wende", "Stralsund, Miltzow", "Stralsund, Klausdorf", "Stralsund, Jaromastraße", "Stralsund, Hexenplatz P+R", "Stralsund, Herzfeld"}

// add searchStopName and replace elements only in lower case
var searchStopName = []string{"l.-feuchtwanger", "-", "/", ",", "ä", "ö", "ü", "ß", "(", ")", ".", "strasse", "haupthst", "wpl", "krhs"}
var replaceStopName = []string{"lion-feuchtwanger", " ", " ", "", "ae", "oe", "ue", "ss", "", "", "", "str", "haupthaltestelle", "wendeplatz", "krankenhaus"}
