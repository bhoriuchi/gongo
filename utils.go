package gongo

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// ToSnakeCase converts to snake case
// https://gist.github.com/stoewer/fbe273b711e6a06315d19552dd4d33e6
func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// GetElement gets the value
func getElement(value interface{}) reflect.Value {
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}
	return rv
}

// GetKind strips the pointer and interface from the kind
func getKind(value interface{}) reflect.Kind {
	rv := getElement(value)
	return rv.Kind()
}

// GetType gets the type using reflect
func getType(value interface{}) reflect.Type {
	rv := getElement(value)
	return reflect.TypeOf(rv)
}

func getTypeName(myvar interface{}) string {
	t := reflect.TypeOf(myvar)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// IsString check if the interface is a string
func isString(value interface{}) bool {
	return getKind(value) == reflect.String
}

// IsMap check if the interface is a map
func isMap(value interface{}) bool {
	return getKind(value) == reflect.Map
}

// IsArrayLike check if the interface is a slice or array
func isArrayLike(value interface{}) bool {
	switch kind := getKind(value); kind {
	case reflect.Slice, reflect.Array:
		return true
	default:
		return false
	}
}

// IsEmpty value is empty
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	} else if isString(value) && fmt.Sprintf("%v", value) == "" {
		return true
	}
	return false
}

// IsNumberLike check if interface is an int
func isNumberLike(value interface{}) bool {
	switch kind := getKind(value); kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// IsIntLike check if interface is an int
func isIntLike(value interface{}) bool {
	switch kind := getKind(value); kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

// Mapify turns an interface into a map
func mapify(src interface{}) (map[string]interface{}, error) {
	m := make(map[string]interface{})

	if !isMap(src) {
		return m, fmt.Errorf("invalid map")
	}

	if err := toInterface(src, &m); err != nil {
		return m, err
	}

	return m, nil
}

// Arrayify turns an interface into a map
func arrayify(src interface{}) ([]interface{}, error) {
	m := make([]interface{}, 0)

	// make array if not an array
	if !isArrayLike(src) {
		a := make([]interface{}, 0)
		src = append(a, src)
	}

	if err := toInterface(src, &m); err != nil {
		return m, err
	}

	return m, nil
}

// ToInterface converts one interface to another using json
func toInterface(src interface{}, dest interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &dest)
}

// IterateeFunc function for iterating
type iterateeFunc func(value interface{}, key interface{}) (interface{}, error)

// DeepMapModify modifys map using key and value modifier functions
func deepMapModify(obj interface{}, valueFunc iterateeFunc, keyFunc iterateeFunc) (interface{}, error) {
	// for arrays loop through each array and make a recursive call
	if isArrayLike(obj) {
		a, err := arrayify(obj)
		if err != nil {
			return nil, err
		}
		result := make([]interface{}, len(a))
		for i, v := range a {
			value, err := deepMapModify(v, valueFunc, keyFunc)
			if err != nil {
				return nil, err
			}
			finalValue, err := valueFunc(value, i)
			if err != nil {
				return nil, err
			}
			result[i] = finalValue
		}
		return result, nil
	}

	// check if the value is not a map, which means that it is a leaf
	// and its value should be returned
	if !isMap(obj) {
		return obj, nil
	}

	// if the value is a map, iterate through the keys and call the function
	m, err := mapify(obj)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for k, v := range m {
		value, err := deepMapModify(v, valueFunc, keyFunc)
		if err != nil {
			return nil, err
		}

		finalValue, err := valueFunc(value, k)
		if err != nil {
			return nil, err
		}

		// NIX this ->> remap the key first in the event it is an object id field
		finalKey, err := keyFunc(finalValue, k)
		if err != nil {
			return nil, err
		}

		if finalKey != nil {
			result[fmt.Sprintf("%v", finalKey)] = finalValue
		}
	}
	return result, nil
}

// Includes determines if an array includes a value
func includes(array interface{}, value interface{}) bool {
	a, err := arrayify(array)
	if err != nil {
		return false
	}
	for _, v := range a {
		if reflect.DeepEqual(v, value) {
			return true
		}
	}
	return false
}

// MapFields maps fields
func mapFields(obj interface{}, filterFunc func(name string, value interface{}) bool) map[string]interface{} {
	result := make(map[string]interface{})
	v := reflect.ValueOf(obj).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.CanInterface() {
			name := t.Field(i).Tag.Get("name")
			value := field.Interface()

			if filterFunc(name, value) {
				result[name] = value
			}
		}
	}

	return result
}

// FilterEmpty removes empty fields from an object and returns a map
func filterEmpty(obj interface{}) map[string]interface{} {
	return mapFields(obj, func(name string, value interface{}) bool {
		return value != nil && fmt.Sprintf("%v", value) != "" && name != ""
	})
}

// tests if obj is an object id
func isObjectID(obj interface{}) bool {
	return getElement(obj).Type() == reflect.TypeOf(primitive.ObjectID([12]byte{}))
}
