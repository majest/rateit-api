package db

import (
	"encoding/json"
	log "github.com/cihub/seelog"
	"reflect"
	"time"
)

type Dto interface {
	Map(object interface{})
}

func C(collection string) *Collection {
	return NewCollection(collection)
}

func ToMap(payload []byte) map[string]interface{} {

	result := make(map[string]interface{})
	err := json.Unmarshal(payload, &result)
	if err != nil {
		panic(err.Error())
	}
	return result
}

func GetParams(payload []byte) Data {
	data := &Data{}
	json.Unmarshal(payload, data)
	return *data
}

func Find(collection string, dto Dto, params map[string]interface{}) error {

	result, err := C(collection).FindOne(params)

	if err != nil {
		return err
	}

	if dto != nil {
		dto.Map(result)
	}
	return nil
}

func FindById(collection string, dto Dto, id string) error {

	params := make(map[string]interface{})
	params["_id"] = id

	result, err := C(collection).FindOne(params)

	if err != nil {
		return err
	}

	if dto != nil {
		dto.Map(result)
	}

	return nil
}

func FindAll(collection string, dto Dto, params map[string]interface{}) error {

	//hasDeleted(dto)
	result, err := C(collection).FindAll(params)

	if err != nil {
		return err
	}

	if dto != nil {
		dto.Map(result)
	}

	return nil
}

func FindAllSorted(collection string, dto Dto, params map[string]interface{}, sort string) error {

	result, err := C(collection).FindAllSorted(params, sort)

	if err != nil {
		return err
	}

	if dto != nil {
		dto.Map(result)
	}
	return nil
}

func Save(collection string, dto Dto, params map[string]interface{}) error {

	// if this element has deleted in the object and we are saving new element,
	// add default delete = false element
	if _, ok := params["_id"]; !ok {

		if hasField(dto, "Deleted") {
			log.Debugf("[DB] Setting deleted : false")
			params["deleted"] = false
		}

		if hasField(dto, "Active") {
			log.Debugf("[DB] Setting active : true")
			params["active"] = true
		}

		if hasField(dto, "Created") {
			params["created"] = time.Now()
		}
	}

	err := C(collection).Save(params, dto)

	if err != nil {
		return err
	}

	return nil
}

func Upsert(collection string, search, params Data) error {
	return C(collection).Upsert(search, params)
}

// Generate object atributes
func hasField(obj interface{}, elem string) bool {

	if obj == nil {
		return false
	}

	typ := reflect.TypeOf(obj).Elem()
	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		if p.Name == elem {
			return true
		}
	}
	return false
}
