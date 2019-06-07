package gongo

import (
	"reflect"

	"github.com/bhoriuchi/gongo/helpers"
)

// Schema creates a new schema
type Schema struct {
	typeName   string
	ref        interface{}
	refType    reflect.Type
	virtuals   *VirtualFieldMap
	middleware *middlewareConfig
}

// NewSchema creates a new schema
func NewSchema(referenceType interface{}) *Schema {
	if referenceType == nil {
		return nil
	}

	refType := helpers.GetType(referenceType)
	typeName := helpers.GetTypeName(referenceType)

	schema := &Schema{
		typeName: typeName,
		ref:      referenceType,
		refType:  refType,
		virtuals: &VirtualFieldMap{},
		middleware: &middlewareConfig{
			pre:  make(map[int]*PreMiddleware),
			post: make(map[int]*PostMiddleware),
		},
	}

	return schema
}
