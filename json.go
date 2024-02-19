package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
)

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
	b, err := os.ReadFile(jsonFilePath)
	if err != nil {
		if *debug {
			log.Println("readCurrentJSON: error while os.ReadFile", err)
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
	err = os.WriteFile(jsonFilePath, b, 0644)
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
