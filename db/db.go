package db

import (
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"reflect"
	"strconv"
	"time"
)

var session *mgo.Session
var initialized bool = false

func GetSession() *mgo.Session {

	if initialized == false {
		InitSession()
	}
	return session
}

func Close() {
	log.Debugf("[Db]  Closing MongoDB session")
	session.Close()
}

func Db() *mgo.Database {
	dbname := "ratings"

	if initialized == false {
		InitSession()
	}

	return session.DB(dbname)
}

func InitSession() {

	if !initialized {

		var err error
		hostname := "localhost"
		dbname := "ratings"
		port := "27017"

		log.Debugf("[DB] Initialising MongoDB session at %s, db: %s, port %s", hostname, dbname, port)

		session, err = mgo.Dial(fmt.Sprintf("%s:%s", hostname, port))
		session.SetSafe(&mgo.Safe{})

		if err != nil {
			panic(fmt.Sprintf("ERROR WHILE CONNECTING TO MONGODB: %s", err))
		}

		initialized = true

	}

}

type Data map[string]interface{}

type Collection struct {
	collectionName string
}

func NewCollection(collectionName string) *Collection {
	return &Collection{collectionName}
}

func (self *Collection) C() *mgo.Collection {
	return Db().C(self.collectionName)
}

func (self *Collection) getConstrains(params Data) (skip int, limit int) {

	if skipParam, ok := params["skip"]; ok {

		skipValue, err := strconv.ParseInt(skipParam.(string), 10, 0)

		if err != nil {
			skip = 0
		}

		skip = int(skipValue)

		delete(params, "skip")

	} else {
		skip = 0
	}

	if limitParam, ok := params["limit"]; ok {

		limitValue, err := strconv.ParseInt(limitParam.(string), 10, 0)

		if err != nil {
			limit = 0
		}

		limit = int(limitValue)
		delete(params, "limit")
	} else {
		limit = 0
	}

	return skip, limit
}

func checkParamsForAdvancedUpdate(params Data) bool {

	for k, _ := range params {
		if k[:1] == "$" {
			return true
		}
	}

	return false
}

func (self *Collection) getSort(params Data) string {
	if sort, ok := params["sort"]; ok {
		delete(params, "sort")
		return sort.(string)
	}

	return ""
}

func (self *Collection) FindOne(params Data) (result Data, err error) {

	now := time.Now()
	if params["_id"] != nil {
		id := params["_id"]
		str, ok := id.(string)

		if !ok {
			log.Debugf("Could not convert to string fuck %s %v", id, id)
		}

		res := make(Data)

		if len(str) != 24 {
			return nil, errors.New("[DB] Invalid Id")
		}

		err = self.C().Find(bson.M{"_id": bson.ObjectIdHex(str)}).One(&res)

		if res["_id"] != "" {
			log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.FindOne : %+v - 1 result", time.Since(now), self.collectionName, params)
		} else {
			log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.FindOne : %+v - 0 results", time.Since(now), self.collectionName, params)
		}

		return res, err
	}

	res := make(Data)

	err = self.C().Find(params).One(&res)

	log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.FindOne : %+v", time.Since(now), self.collectionName, params)
	return res, err
}

func (self *Collection) FindAll(params Data) (result []Data, err error) {
	now := time.Now()

	params = self.replaceId(params)
	skip, limit := self.getConstrains(params)
	sort := self.getSort(params)

	if sort == "" {
		sort = "+order"
	}

	err = self.C().Find(params).Sort(sort).Skip(skip).Limit(limit).All(&result)

	if len(result) > 500 {
		return nil, errors.New("Results exceeds 500 records. Please use skip and limit")
	}

	log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.FindAll : %#v - %v results", time.Since(now), self.collectionName, params, len(result))
	return result, err
}

func (self *Collection) FindAllSorted(params Data, sort string) (result []Data, err error) {

	now := time.Now()
	params = self.replaceId(params)
	skip, limit := self.getConstrains(params)

	err = self.C().Find(params).Sort(sort).Skip(skip).Limit(limit).All(&result)

	if len(result) > 500 {
		return nil, errors.New("Results exceeds 500 records. Please use skip and limit")
	}

	log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.FindAllSorted : %+v", time.Since(now), self.collectionName, params)
	return result, err
}

func (self *Collection) Delete(id string) error {

	err := self.C().RemoveId(bson.ObjectIdHex(id))
	if err != nil {
		log.Errorf("[DB] Could not delete: %s", err.Error())
		return err
	}
	return nil
}

func (self *Collection) Save(params Data, dto Dto) error {

	now := time.Now()

	var err error
	if params["_id"] != nil {

		// update
		id := params["_id"]
		delete(params, "_id")

		paramId, err := self.getId(id)

		if err != nil {
			return err
		}

		if dto != nil {
			dto.Map(params)
		}

		if checkParamsForAdvancedUpdate(params) {
			err = self.C().Update(paramId, params)
		} else {
			err = self.C().Update(paramId, bson.M{"$set": params})
		}

		if err != nil {
			log.Errorf("[DB][Error] Could not update: %s", err.Error())
		}

	} else {

		// insert
		id := bson.NewObjectId()
		params["_id"] = id

		if dto != nil {
			dto.Map(params)
		}

		err = self.C().Insert(params)
		log.Debugf("[DB] Insert: %#s in %s", params, self.collectionName)

	}

	if err != nil {
		log.Errorf("[DB] Could not save: %s", err.Error())
		return err
	}

	log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.Save", time.Since(now), self.collectionName)
	return nil
}

func (self *Collection) replaceId(params Data) Data {
	if params["_id"] != nil {

		bsonId, err := self.getId(params["_id"])
		if err != nil {
			return params
		}
		params["_id"] = bsonId["_id"]
	}
	return params
}

// get id basing on type
func (self *Collection) getId(value interface{}) (Data, error) {

	if value != nil {

		log.Debugf("[DB] Id is %s", reflect.TypeOf(value).Kind().String())

		objectType := reflect.TypeOf(value).String()

		// if it's a string just return id
		if objectType == "string" {

			if len(value.(string)) != 24 {
				return nil, errors.New("[DB] Invalid Id")
			}
			return Data{"_id": bson.ObjectIdHex(value.(string))}, nil

		} else if objectType == "map[string]interface {}" {

			array := value.(map[string]interface{})
			return self.getIdSet(array), nil

		} else if objectType == "db.Data" {

			array := value.(Data)
			return self.getIdSet(array), nil
		}

	}

	return nil, errors.New("[DB] Bad _id format")
}

// get the set of ids for Data
func (self *Collection) getIdSet(array Data) Data {

	params := []bson.ObjectId{}

	if array["$in"] != nil {

		for _, v := range array["$in"].([]string) {
			params = append(params, bson.ObjectIdHex(v))
		}
		return Data{"_id": Data{"$in": params}}
	}

	return Data{}
}

func (self *Collection) Upsert(search, params Data) error {
	_, err := self.C().Upsert(search, params)
	return err
}
