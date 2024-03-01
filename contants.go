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
const templateFileEnding = ".go.tmpl"
const templateName = "haltestellenabgleich"
const tmplDirectory = "tmpl"
const vvrDataFile = "vvr.json"
const vvrSearchURL = "https://vvr.verbindungssuche.de/fpl/suhast.php?&query="

// a bbox would be (53.948,12.2189,54.6937,13.7892)
// normal Overpass-Turbo query:
//
//	[out:json][timeout:600];({{geocodeArea:"Vorpommern-Rügen"}};{{geocodeArea:"Graal-Müritz"}};{{geocodeArea:"Greifswald"}};)->.searchArea;(nw["public_transport"="platform"]["bus"](area.searchArea);node["public_transport"="stop_position"]["bus"](area.searchArea);node["highway"="bus_stop"](area.searchArea);rel["type"="public_transport"]["bus"](area.searchArea););out;
//
// needs to be converted to Overpass-API query (compact variant):
//
//	[out:json][timeout:600];(area(3601739379);area(3600393349);area(3600062363);)->.searchArea;(nwr["public_transport"="platform"]["bus"](area.searchArea);node["public_transport"="stop_position"]["bus"](area.searchArea);node["highway"="bus_stop"](area.searchArea);relation["type"="public_transport"]["bus"](area.searchArea););out;
//
// Vorpommern-Rügen = 3601739379
// Graal-Müritz = 3600393349
// Greifswald = 3600062363
// Landhagen =  3601432580 // rund um Greifswald
const overpassSearchArea = "area(3601739379);area(3600393349);area(3600062363);area(3601432580);"
const overpassURL = "http://overpass-api.de/api/interpreter?data="
const overpassQueryPrefix = "[out:json][timeout:600];("
const overpassQuerySuffix = ")->.searchArea;(nw[\"public_transport\"=\"platform\"][\"bus\"](area.searchArea);node[\"public_transport\"=\"stop_position\"][\"bus\"](area.searchArea);node[\"highway\"=\"bus_stop\"](area.searchArea);rel[\"type\"=\"public_transport\"][\"bus\"](area.searchArea););out;"

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
	"Darßbahn":                          0,
	"Rügen-Bahnen":                      0,
	"Kap-Arkona-Bahn":                   0,
	"Busunternehmen M. Scholz":          0,
	"Deutsche Bahn":                     0,
	"Flixbus":                           0,
	"Gemeinde Binz":                     0,
	"Regenbogencamp Nonnevitz":          0,
	"Stadtwerke Greifswald GmbH (SWG)":  0,
	"Stadtwerke Greifswald GmbH":        0,
	"Anklamer Verkehrsgesellschaft mbH": 0,
}
var httpClient = &http.Client{Timeout: 1000 * time.Second}
var ignoreVvrStops = []string{"Richtenberg, Mühlenbergstraße", "Kölzow", "Kloster, Kirchweg", "Neu Lüdershagen, II", "Vitte, Hafen", "Klevenow, Gemeinde", "Stralsund, Franzburg", "Schulenberg, Feuerwehr", "Papenhagen, Ersatzhaltestelle", "Hoikenhagen, Ersatzhaltestelle", "Stralsund, Bremer Str.", "Bergen, Industriestraße", "Barth, Vineta Sportarena", "Baabe, Haus des Gastes", "Baabe, Göhrener Chaussee", "Lassentin, Ausbau Ort", "Kölzow, Ausbau", "Richtenberg, Am Sportplatz", "Poggendorf, Alte Dorfstraße", "Stralsund, Altenpleen", "Stralsund, Betriebsfahrt", "Glowe, Wendeplatz", "Gager, Hafen", "Franzburg, Garthofstraße", "Dorow, Abzweig", "Damgarten, Bahnhof Ost", "Camper, Ortseingang", "Camitz, Försterei", "Balkenkoppel, Abzweig", "Groß Lehmhagen, Dorf", "Stralsund, Krönnevitz", "Stralsund, Kummerow", "SEV", "(Workshop)", "Schulbus", "Wagen defekt", "Stralsund, Velgast", "Stralsund, Tribseer Wiesen", "Sonderfahrt", "Probefahrt", "Stralsund, O.-Palme-Platz Wende", "Stralsund, Miltzow", "Stralsund, Klausdorf", "Stralsund, Jaromastraße", "Stralsund, Hexenplatz P+R", "Stralsund, Herzfeld"}

// add searchStopName and replace elements only in lower case
var searchStopName = []string{"straße d. jugend", "elmemhorst", "gr.", "lüdershg.", "bartelshg.ii", "c.-heydemann", "h.-von-stephan", "e.-m.-arndt", "h.-heine-ring", "deutsche rentenversicherung", "l.-feuchtwanger", "-", "/", ",", "ä", "ö", "ü", "ß", "(", ")", ".", "strasse", "haupthst", "wpl", "krhs"}
var replaceStopName = []string{"straße der jugend", "elmenhorst", "groß", "lüdershagen", "bartelshagen ii", "carl-heydemann", "heinrich-von-stephan", "ernst-moritz-arndt", "heinrich-heine-ring", "drv", "lion-feuchtwanger", " ", " ", "", "ae", "oe", "ue", "ss", "", "", "", "str", "haupthaltestelle", "wendeplatz", "krankenhaus"}
