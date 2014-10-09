package main

import (
	"encoding/json"
	"fmt"
	//"fmt"
	"bitbucket.org/arturgrabowski/go-service-layer/stats"
	log "github.com/cihub/seelog"
	"github.com/majest/rateit-api/db"
	"github.com/majest/rateit-api/parser"
	"io/ioutil"
	"net/http"
	"time"
)

var collection *db.Collection

type RatingRequest struct {
	Uid  string `json:"uid"`
	Url  string `json:"url"`
	Data Rating `json:"data"`
}

type ErrorResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

type Response struct {
	Status string `json:"status"`
}

type Rating struct {
	Rating map[string]int `json:"rating"`
	Time   time.Time      `json:"date"`
}

func (el *Rating) ToMap() map[string]interface{} {
	mappedValues := make(map[string]interface{})
	bytes, _ := json.Marshal(el)
	json.Unmarshal(bytes, &mappedValues)
	return mappedValues
}

type SiteEntry struct {
	Id     string       `json:"_id,omitempty"`
	Url    string       `json:"url,omitempty"`
	Rating []Rating     `json:"ratings,omitempty"`
	Site   *parser.Site `json:"site,omitempty"`
}

func (p *SiteEntry) Map(object interface{}) {
	bytes, _ := json.Marshal(object)
	json.Unmarshal(bytes, p)
}

func (el *SiteEntry) ToMap() map[string]interface{} {
	mappedValues := make(map[string]interface{})
	bytes, _ := json.Marshal(el)
	json.Unmarshal(bytes, &mappedValues)
	return mappedValues
}

func main() {
	collection = db.NewCollection("sites")
	db.InitSession()
	serve()
}

func handler(w http.ResponseWriter, r *http.Request) {
	bodyRequest, err := ioutil.ReadAll(r.Body)

	// set the initial header
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err != nil {
		stats.Counter(1.0, "com.arnet.rateit.error.reading", 1)
		msg := fmt.Sprintf("Error while reading body: %s", err.Error())
		log.Errorf(msg)
		w.WriteHeader(http.StatusInternalServerError)
		responseError(msg)
		return
	}

	// rating request
	ratingRequest := &RatingRequest{}
	err = json.Unmarshal(bodyRequest, ratingRequest)

	if err != nil {
		stats.Counter(1.0, "com.arnet.rateit.error.unmarshal", 1)
		msg := fmt.Sprintf("Error while decoding body: %s. Body: %s", err.Error(), bodyRequest)
		w.WriteHeader(http.StatusBadRequest)
		log.Errorf(msg)
		responseError(msg)
		return
	}

	response()
	go saveRating(ratingRequest)
}

func responseError(message string) []byte {
	resp, _ := json.Marshal(&ErrorResponse{Error: true, Message: message})
	return resp
}

func response() []byte {
	resp, _ := json.Marshal(&Response{Status: "ok"})
	return resp
}

func saveRating(ratingRequest *RatingRequest) {

	ratingRequest.Data.Time = time.Now()

	// check if we have url
	if ratingRequest.Url != "" {

		siteEntry := &SiteEntry{}
		err := db.Find("ratings", siteEntry, db.Data{"url": ratingRequest.Url})

		// new entry
		if err != nil {

			site := parser.New(ratingRequest.Url)
			site.Parse()

			// prepare site
			siteEntry.Rating = []Rating{ratingRequest.Data}
			siteEntry.Url = ratingRequest.Url
			siteEntry.Site = site

			db.Save("ratings", siteEntry, siteEntry.ToMap())

			stats.Counter(1.0, "com.arnet.rateit.entry.create", 1)

		} else {

			// just pushing new rating
			siteEntry.Rating = append(siteEntry.Rating, ratingRequest.Data)
			db.Save(
				"ratings",
				siteEntry,
				siteEntry.ToMap(),
			)

			stats.Counter(1.0, "com.arnet.rateit.entry.update", 1)
		}
	} else {
		stats.Counter(1.0, "com.arnet.rateit.error.missingurl", 1)
	}
}

func serve() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":9010", nil)
}
