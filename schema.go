package gongo

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/mitchellh/mapstructure"

	"github.com/bhoriuchi/gongo/helpers"
	"go.mongodb.org/mongo-driver/bson"
)

// Built-in types
const (
	StringType = "String"
	IntType    = "Int"
	FloatType  = "Float"
	BoolType   = "Bool"
	MixedType  = "Mixed"
	ObjectID   = "ObjectID"
)

var intRx = regexp.MustCompile(`^\d+$`)

// Schema a schema definition
type Schema struct {
	Fields      SchemaFieldMap
	Options     *SchemaOptions
	Virtuals    *VirtualFieldMap
	middleware  *middlewareConfig
	initialized bool
}

func (c *Schema) init() error {
	if c.initialized {
		return nil
	}
	for name, field := range c.Fields {
		if field == nil {
			return fmt.Errorf("definition for schema field %q cannot be nil", name)
		}
		if err := field.init(name); err != nil {
			return err
		}
	}
	c.initialized = true
	return nil
}

// creates a copy of a schema
func (c *Schema) copy() *Schema {
	var options SchemaOptions
	var virtuals VirtualFieldMap
	var middleware middlewareConfig

	if c.Options != nil {
		options = c.Options.copy()
	} else {
		options = SchemaOptions{}
	}
	if c.Virtuals != nil {
		virtuals = c.Virtuals.copy()
	} else {
		virtuals = VirtualFieldMap{}
	}
	if c.middleware != nil {
		middleware = c.middleware.copy()
	} else {
		middleware = middlewareConfig{}
	}

	newSchema := Schema{
		Fields:     c.Fields.copy(),
		Options:    &options,
		Virtuals:   &virtuals,
		middleware: &middleware,
	}
	return &newSchema
}

// SchemaField a schema field definition
type SchemaField struct {
	Type        interface{}
	Required    bool
	Unique      bool
	Default     interface{}
	Validate    *[]ValidatorFunc
	Meta        *map[string]interface{}
	elementType interface{}
	isArray     bool
}

// initializes a schema field
func (c *SchemaField) init(name string) error {
	if c.Type == nil {
		return fmt.Errorf("field %q cannot be of type nil", name)
	} else if name == "" {
		return fmt.Errorf("empty field names are not allowed")
	}

	// determine if an array
	switch kind := helpers.GetKind(c.Type); kind {
	case reflect.Slice, reflect.Array:
		s := helpers.GetElement(c.Type)
		if s.Len() != 1 {
			return fmt.Errorf("array type definitions require exactly one type enclosed in an array")
		}
		c.isArray = true
		c.elementType = s.Index(0).Interface()
	default:
		c.isArray = false
		c.elementType = c.Type
	}

	// determine if element type is a valid one
	if helpers.GetElement(c.elementType).Type() == reflect.TypeOf(Schema{}) {
		// if the element is a schema, try to initialize it
		schema := helpers.GetElement(c.elementType).Interface().(Schema)
		schema.init()
		return nil
	}

	switch c.elementType {
	case StringType, IntType, FloatType, BoolType, MixedType:
		return nil
	}

	return fmt.Errorf("field %q has an invalid type defined", name)
}

// Copies a schema field
func (c *SchemaField) copy() *SchemaField {
	validators := make([]ValidatorFunc, 0)
	if c.Validate != nil {
		for _, fn := range *c.Validate {
			validators = append(validators, fn)
		}
	}
	meta := make(map[string]interface{})
	if c.Meta != nil {
		for k, v := range *c.Meta {
			meta[k] = v
		}
	}

	newField := SchemaField{
		Type:        c.Type,
		Required:    c.Required,
		Unique:      c.Unique,
		Default:     c.Default,
		Validate:    &validators,
		Meta:        &meta,
		elementType: c.elementType,
		isArray:     c.isArray,
	}
	return &newField
}

// SchemaFieldMap a map of fields
type SchemaFieldMap map[string]*SchemaField

func (c *SchemaFieldMap) copy() SchemaFieldMap {
	fields := make(SchemaFieldMap)
	for k, v := range *c {
		fields[k] = v.copy()
	}
	return fields
}

// ValidatorFunc a function that performs a validation
type ValidatorFunc func(value interface{}) error

// SchemaOptions schema options
type SchemaOptions struct {
	ID *bool
}

func (c *SchemaOptions) copy() SchemaOptions {
	options := SchemaOptions{
		ID: c.ID,
	}
	return options
}

func (c *Schema) prepareData(doc interface{}, isQuery bool) *bson.M {
	result := bson.M{}
	document := bson.M{}

	// ensure that the doc is a document
	switch kind := helpers.GetKind(doc); kind {
	case reflect.Map:
		if err := mapstructure.WeakDecode(doc, &document); err != nil {
			return &document
		}
	default:
		return &document
	}

	for name, field := range c.Fields {
		if value, hasField := document[name]; hasField && value != nil {
			// for mixed non-array values, return right away. this will handle non-schema maps
			if field.elementType == MixedType && !field.isArray {
				result[name] = value
				continue
			}

			// handle object id types first since they are byte arrays
			if helpers.IsObjectID(value) {
				if field.elementType == ObjectID && field.isArray == false {
					result[name] = value
				}
				continue
			}

			el := helpers.GetElement(field.elementType)
			fieldIsSchema := el.Type() == reflect.TypeOf(Schema{})

			switch kind := helpers.GetKind(value); kind {

			// handle arrays
			case reflect.Array, reflect.Slice:
				if !field.isArray {
					continue
				}
				if !fieldIsSchema {
					result[name] = value
					continue
				}
				arrayValue := make([]interface{}, 0)
				fieldSchema := el.Interface().(Schema)
				arrayEl := helpers.GetElement(value)
				for i := 0; i < arrayEl.Len(); i++ {
					itemValue := fieldSchema.prepareData(arrayEl.Index(i).Interface(), isQuery)
					if itemValue != nil {
						arrayValue = append(arrayValue, itemValue)
					}
				}
				result[name] = arrayValue

			// handle maps, which should be a schema
			case reflect.Map:
				if !fieldIsSchema {
					continue
				}
				fieldSchema := el.Interface().(Schema)
				mapValue := fieldSchema.prepareData(value, isQuery)
				if mapValue != nil {
					result[name] = mapValue
				}

			default:
				result[name] = value
			}
		}
	}

	// apply setters
	finalResult, err := c.applyVirtualSetters(result)
	if err != nil {
		return &bson.M{}
	}
	return finalResult
}

// returns a copy of the document with undefined fields removed
func (c *Schema) copyInternalDocument(doc bson.M) bson.M {
	newDoc := bson.M{}

	for k, v := range doc {
		if _, ok := c.Fields[k]; ok || k == "_id" {
			newDoc[k] = v
		}
	}

	return newDoc
}

// adds default values for missing fields
func (c *Schema) setDefaults(doc bson.M) {
	for name, field := range c.Fields {
		if _, ok := doc[name]; !ok && field.Default != nil {
			doc[name] = field.Default
		}
	}
}

// checks if a schema has a specified field path
func (c *Schema) hasFieldPath(fieldPath []string) bool {
	if len(fieldPath) == 0 {
		return false
	}
	p := fieldPath[0]

	field, hasField := c.Fields[p]
	if hasField {
		if len(fieldPath) == 1 {
			return true
		}
		remaining := fieldPath[1:]

		if field.isArray {
			i := remaining[0]
			if !intRx.MatchString(i) || len(remaining) < 2 {
				return false
			}
			remaining = remaining[1:]
		}

		el := helpers.GetElement(field.elementType)
		if el.Type() != reflect.TypeOf(Schema{}) {
			return false
		}
		s := el.Interface().(Schema)
		return s.hasFieldPath(remaining)
	}

	return false
}
