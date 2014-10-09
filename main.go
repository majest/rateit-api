package main

import (
	"encoding/json"
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

	if err != nil {
		stats.Counter(1.0, "com.arnet.rateit.error.reading", 1)
		log.Errorf("Error while reading body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// rating request
	ratingRequest := &RatingRequest{}
	err = json.Unmarshal(bodyRequest, ratingRequest)

	if err != nil {
		stats.Counter(1.0, "com.arnet.rateit.error.unmarshal", 1)
		log.Errorf("Error while decoding body: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte("{\"ok\":1}"))
	go saveRating(ratingRequest)
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
