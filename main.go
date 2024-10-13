package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type LatLon struct {
	LAT string `json:"lat"`
	LON string `json:"lon"`
}

func main() {
	var latlonData []LatLon
	UnmarshalJSON(GetLatLon(), &latlonData)

	fmt.Printf("Test for lat lon data: %v\n", latlonData[0])

	//fmt.Printf("Lat for your zip is: %s\n", latlonData.LAT)
	//fmt.Printf("Lon for your zip is: %s\n", latlonData.LON)
}

func GetLatLon() []byte {
	var zipCode string
	fmt.Printf("Please enter your zip code: ")
	fmt.Scanln(&zipCode)

	url := fmt.Sprintf("https://nominatim.openstreetmap.org/search?postalcode=%s&country=United%20States&format=json", zipCode)
	client := &http.Client{}

	var htmlData []byte
	for i := 0; i < 2; i++ {
		req, reqErr := http.NewRequest("GET", url, nil)

		if reqErr != nil {
			PrintRetry(i, reqErr)
			continue
		}

		req.Header.Set("User-Agent", "gowx")

		httpResp, httpErr := client.Do(req)

		if httpErr != nil {
			PrintRetry(i, httpErr)
			continue
		}

		if httpResp.StatusCode != 200 {
			log.Println("HTTP error code: ", httpResp.StatusCode)
			htmlData, htmlErr := ioutil.ReadAll(httpResp.Body)
			if htmlErr != nil {
				PrintRetry(i, htmlErr)
			}
			htmlErr = errors.New(string(htmlData))
			PrintRetry(i, htmlErr)
			continue
		}
		defer httpResp.Body.Close()

		htmlData, htmlErr := ioutil.ReadAll(httpResp.Body)
		if htmlErr != nil {
			PrintRetry(i, htmlErr)
		}
		fmt.Printf(string(htmlData))
		return htmlData
	}
	return htmlData
}

func UnmarshalJSON(htmlData []byte, latlonData *[]LatLon) {
	jsonErr := json.Unmarshal(htmlData, &latlonData)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}
}

func PrintRetry(tries int, err error) {
	if tries == 0 {
		log.Println(err)
		fmt.Printf("Will try again in one minute.\n")
	} else {
		log.Fatal("No more tries. ", err)
	}
	time.Sleep(10 * time.Second)
}
