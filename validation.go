package gongo

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bhoriuchi/gongo/helpers"
	"go.mongodb.org/mongo-driver/bson"
)

// checks that required field exists
func (c *Schema) checkRequired(field SchemaField, fieldPath []string, hasField bool) error {
	if field.Required && !hasField {
		return fmt.Errorf("required field %q not specified", strings.Join(fieldPath, "."))
	}
	return nil
}

// validates that the value type is the expected type
func validateType(value, expectedType interface{}, fieldPath []string, skipRequired bool) error {
	// mixed type allows anything
	if expectedType == MixedType {
		return nil
	}
	name := strings.Join(fieldPath, ".")
	switch kind := helpers.GetKind(value); kind {
	case reflect.String:
		if expectedType != StringType {
			return fmt.Errorf("field %q is not a valid %s string", name, expectedType)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if expectedType != IntType {
			return fmt.Errorf("field %q is not a valid %s", name, expectedType)
		}
	case reflect.Float32, reflect.Float64:
		if expectedType != FloatType {
			return fmt.Errorf("field %q is not a valid %s", name, expectedType)
		}
	case reflect.Bool:
		if expectedType != BoolType {
			return fmt.Errorf("field %q is not a valid %s", name, expectedType)
		}
	case reflect.Map:
		// determine if element type is a valid one
		el := helpers.GetElement(expectedType)
		if el.Type() == reflect.TypeOf(Schema{}) {
			// if the element is a schema, try to initialize it
			schema := el.Interface().(Schema)
			schema.init()
			b := bson.M{}
			if err := weakDecode(value, &b); err != nil {
				return fmt.Errorf("field %q is not a valid %s", name, el.Type().Name())
			}
			return schema.validate(&b, fieldPath, skipRequired)
		}
	}
	return nil
}

// validates that the defined schema type is the actual type
func (c *Schema) validateType(field SchemaField, fieldPath []string, value interface{}, skipRequired bool) error {
	// if the field is not an array, validate it
	if !field.isArray {
		return validateType(value, field.elementType, fieldPath, skipRequired)
	}

	// otherwise, iterate though each element and validate it
	s := helpers.GetElement(value)
	for i := 0; i < s.Len(); i++ {
		if err := validateType(
			s.Index(i).Interface(),
			field.elementType,
			append(fieldPath, fmt.Sprintf("%d", i)),
			skipRequired,
		); err != nil {
			return err
		}
	}
	return nil
}

func (c *Schema) validate(doc *bson.M, path []string, skipRequired bool) error {
	if doc == nil {
		return fmt.Errorf("no document to validate")
	}
	document := *doc
	for name, field := range c.Fields {
		fieldPath := append(path, name)
		fieldValue, hasField := document[name]

		// check if the field is required
		if !skipRequired {
			if err := c.checkRequired(*field, fieldPath, hasField); err != nil {
				return err
			}
		}

		// if there is no field, continue to the next
		if !hasField {
			continue
		}

		// otherwise validate the value type
		if err := c.validateType(*field, fieldPath, fieldValue, skipRequired); err != nil {
			return err
		}

		// now perform custom validations if defined
		if field.Validate != nil {
			for _, validator := range *field.Validate {
				if err := validator(fieldValue); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Validate validates a document
func (c *Document) Validate() error {
	return c.model.schema.validate(c.next, []string{}, false)
}

// filters non schema fields out
func (c *Schema) filterUndefined(doc *bson.M) *bson.M {
	result := bson.M{}

	for fieldName, fieldValue := range *doc {
		// immediately add id
		if fieldName == "_id" {
			result[fieldName] = fieldValue
			continue
		}

		field, hasField := c.Fields[fieldName]
		if !hasField {
			continue
		}

		el := helpers.GetElement(field.elementType)
		isNestedSchema := el.Type() == reflect.TypeOf(Schema{})

		// handle object ids, mixed types, and non schema types
		if helpers.IsObjectID(fieldValue) || field.elementType == MixedType || !isNestedSchema {
			result[fieldName] = fieldValue
			continue
		}

		// get the nested schema
		schema := el.Interface().(Schema)
		schema.init()

		// get the data kind
		switch kind := helpers.GetKind(fieldValue); kind {
		case reflect.Array, reflect.Slice:
			// if the data is an array and the field is not, ignore the data
			if !field.isArray {
				continue
			}
			a := make([]interface{}, 0)
			// loop through
			fieldEl := helpers.GetElement(fieldValue)
			for i := 0; i < fieldEl.Len(); i++ {
				m := bson.M{}
				if err := weakDecode(fieldValue, &m); err != nil {
					continue
				}

				nested := schema.filterUndefined(&m)
				if nested != nil {
					a = append(a, nested)
				}
			}

			result[fieldName] = a
		case reflect.Map:
			// if marked as an array ignore
			if field.isArray {
				continue
			}

			m := bson.M{}
			if err := weakDecode(fieldValue, &m); err != nil {
				continue
			}

			nested := schema.filterUndefined(&m)
			if nested != nil {
				result[fieldName] = nested
			}
		}
	}

	return &result
}
