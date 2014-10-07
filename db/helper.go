package db

import (
	"encoding/json"
	"github.com/bitly/go-simplejson"
	"reflect"
)

type Dto interface {
	Map(object interface{})
}

func C(collection string) *Collection {
	return NewCollection(collection)
}

func ToMap(payload []byte) map[string]interface{} {

	json, _ := simplejson.NewJson(payload)
	result, err := json.Map()

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

	result, err := C(collection).Save(params)

	if err != nil {
		return err
	}

	if dto != nil {
		dto.Map(result)
	}
	return nil
}

func Upsert(collection string, search, params Data) error {
	return C(collection).Upsert(search, params)
}

// Generate object atributes
func hasField(obj interface{}, elem string) bool {

	typ := reflect.TypeOf(obj).Elem()
	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		if p.Name == elem {
			return true
		}
	}
	return false
}
