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
	client := &http.Client{}
	//var location LatLon
	var htmlData []byte
	GetLatLon(client, &htmlData)
	fmt.Printf("This is htmlData within the main function: %s\n", string(htmlData))
	/*if latlonErr := json.Unmarshal(htmlData, &location); latlonErr != nil {
		log.Fatal(latlonErr)
	}
	//UnmarshalJSON(GetLatLon(client), &latlonData)

	fmt.Printf("Lat for your zip is: %s\n", location.LAT)
	fmt.Printf("Lon for your zip is: %s\n", location.LON)*/
}

func GetLatLon(client *http.Client, htmlData *[]byte) {
	var zipCode string
	fmt.Printf("Please enter your zip code: ")
	fmt.Scanln(&zipCode)

	//url := fmt.Sprintf("https://nominatim.openstreetmap.org/search?postalcode=%s&country=United%20States&format=json", zipCode)

	url := "https://api.coingecko.com/api/v3/coins/bitcoin-cash"

	for i := 0; i < 2; i++ {
		fmt.Printf("At the top of the for loop. %s\n", htmlData)
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

		htmlData = htmlData
		fmt.Println(i)
		fmt.Scanln()
		break
	}
	fmt.Printf("Within the GetLatLon function: %s\n", *htmlData)
}

func (location *LatLon) UnmarshalJSON(htmlData []byte) error {
	tmp := []interface{}{&location.LAT, &location.LON}
	wantLen := len(tmp)
	if jsonErr := json.Unmarshal(htmlData, &tmp); jsonErr != nil {
		return jsonErr
	}
	if g, e := len(tmp), wantLen; g != e {
		return fmt.Errorf("Wrong number of fields in LatLon: %d != %d.\n", g, e)
	}
	return nil
}

//func UnmarshalJSON(htmlData []byte, latlonData *[]LatLon) {
//	jsonErr := json.Unmarshal(htmlData, &latlonData)
//	if jsonErr != nil {
//		log.Fatal(jsonErr)
//	}
//}

func PrintRetry(tries int, err error) {
	if tries == 0 {
		log.Println(err)
		fmt.Printf("Will try again in one minute.\n")
	} else {
		log.Fatal("No more tries. ", err)
	}
	time.Sleep(10 * time.Second)
}
