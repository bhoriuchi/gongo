package helpers

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// ToSnakeCase converts to snake case
// https://gist.github.com/stoewer/fbe273b711e6a06315d19552dd4d33e6
func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// GetElement gets the value
func GetElement(value interface{}) reflect.Value {
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}
	return rv
}

// GetKind strips the pointer and interface from the kind
func GetKind(value interface{}) reflect.Kind {
	rv := GetElement(value)
	return rv.Kind()
}

// GetType gets the type using reflect
func GetType(value interface{}) reflect.Type {
	rv := GetElement(value)
	return reflect.TypeOf(rv)
}

// GetTypeName gets the type name
func GetTypeName(myvar interface{}) string {
	t := reflect.TypeOf(myvar)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// IsString check if the interface is a string
func IsString(value interface{}) bool {
	return GetKind(value) == reflect.String
}

// ToInterface converts one interface to another using json
func ToInterface(src interface{}, dest interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &dest)
}

// IsObjectID tests if obj is an object id
func IsObjectID(obj interface{}) bool {
	return GetElement(obj).Type() == reflect.TypeOf(primitive.ObjectID([12]byte{}))
}

// DotPathToSlashPath converts a dot path to a dir path
func DotPathToSlashPath(p string) string {
	s := strings.ReplaceAll(p, ".", "/")
	return filepath.Clean(fmt.Sprintf("/%s", s))
}
