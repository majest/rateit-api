package db

import (
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
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

var dbname = "driverBackfill-state"

func Db() *mgo.Database {

	if initialized == false {
		InitSession()
	}

	return session.DB(dbname)
}

func InitSession() {

	if !initialized {

		var err error

		hostname := "localhost"

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
func (self *Collection) getSort(params Data) string {
	if sort, ok := params["sort"]; ok {
		delete(params, "sort")
		return sort.(string)
	}

	return ""
}

func (self *Collection) FindOne(params Data) (result Data, err error) {

	//now := time.Now()
	if params["_id"] != nil {
		id := params["_id"]
		str, ok := id.(string)

		if !ok {
			log.Debugf("Could not convert to string fuck %s %v", id, id)
		}

		res := make(map[string]interface{})

		if len(str) != 24 {
			return nil, errors.New("[DB] Invalid Id")
		}

		err = self.C().Find(bson.M{"_id": bson.ObjectIdHex(str)}).One(&res)

		if res["_id"] != "" {
			//  log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.FindOne : %+v - 1 result", time.Since(now), self.collectionName, params)
		} else {
			//  log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.FindOne : %+v - 0 results", time.Since(now), self.collectionName, params)
		}

		return res, err
	}

	err = self.C().Find(params).One(&result)
	//log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.FindOne : %+v", time.Since(now), self.collectionName, params)
	return result, err
}

//var ids = []string{"53873eed3f75623cbf0005a8", "53873ef33f75623cbf00065d", "53873ef33f75623cbf000667", "53873ef43f75623cbf000671"}

func (self *Collection) FindAll() (result []Data, err error) {

	// bsonids := []bson.ObjectId{}
	// for _, id := range ids {
	//  bsonids = append(bsonids, bson.ObjectIdHex(id))
	// }

	//err = self.C().Find(Data{"_id": Data{"$in": bsonids}}).All(&result)
	err = self.C().Find(nil).All(&result)

	//log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.FindAll : %+v - %v results", time.Since(now), self.collectionName, params, len(result))
	return result, err
}

func (self *Collection) FindAllSorted(params Data, sort string) (result []Data, err error) {

	now := time.Now()
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

func (self *Collection) SaveBy(search Data, params Data) error {
	object := params
	return self.C().Update(search, bson.M{"$set": object})
}

func (self *Collection) Save(params Data) (result Data, err error) {

	//now := time.Now()
	result = Data{}
	object := params

	if object["_id"] != nil && object["_id"].(string) != "" {

		// update
		id := object["_id"].(string)
		result["_id"] = id
		delete(object, "_id")

		if len(id) != 24 {
			return nil, errors.New("[DB] Invalid Id")
		}

		err = self.C().UpdateId(bson.ObjectIdHex(id), bson.M{"$set": object})
		//log.Debugf("[DB] Saving by id: %s in %s", id, self.collectionName)

		if err != nil {
			log.Errorf("[DB][Error] Could not update: %s", err.Error())
		}

	} else {

		// insert
		idHex := bson.NewObjectId()
		object["_id"] = idHex
		id := idHex.Hex()

		result["_id"] = id
		err = self.C().Insert(object)

		//log.Debugf("[DB] Insert: %s in %s", object, self.collectionName)
	}

	if err != nil {
		log.Errorf("[DB] Could not save: %s", err.Error())
		return nil, err
	}

	//log.Debugf("[DB] \x1b[31;1m%v\x1b[0m %s.Save : %+v", time.Since(now), self.collectionName, object)
	return result, nil
}

func (self *Collection) Upsert(search, params Data) error {
	_, err := self.C().Upsert(search, params)
	return err
}
