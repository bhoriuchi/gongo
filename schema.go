package gongo

import (
	"fmt"
	"reflect"

	"github.com/bhoriuchi/gongo/helpers"
)

// Schema creates a new schema
type Schema struct {
	typeName    string
	ref         interface{}
	refType     reflect.Type
	fieldTagDef *FieldTagDefinition
	virtuals    *VirtualFieldMap
}

// NewSchema creates a new schema
func NewSchema(referenceType interface{}) *Schema {
	if referenceType == nil {
		return nil
	}

	refType := helpers.GetType(referenceType)
	typeName := helpers.GetTypeName(referenceType)

	schema := &Schema{
		typeName:    typeName,
		ref:         referenceType,
		refType:     refType,
		fieldTagDef: DefaultFieldTagDefinition(),
		virtuals:    &VirtualFieldMap{},
	}

	return schema
}

// WithFieldTags sets the field tag definition to use
// this is useful if the default tags conflict with other packages
func (c *Schema) WithFieldTags(definition *FieldTagDefinition) error {
	if definition == nil {
		return fmt.Errorf("no tag definition specified")
	} else if err := definition.Validate(); err != nil {
		return err
	}
	c.fieldTagDef = definition
	return nil
}
