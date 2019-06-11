package gongo

import (
	"fmt"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bhoriuchi/gongo/helpers"
)

type walkOptions struct {
	applySetters     bool
	applyDefaults    bool
	castObjectID     bool
	validateTypes    bool
	validateCustom   bool
	validateRequired bool
}

// walk walks a schema performing the requested operations
func (c *Schema) walk(value interface{}, path []string, options *walkOptions) (*bson.M, error) {
	output := bson.M{}
	document := bson.M{}
	if err := c.gongo.weakDecode(value, &document); err != nil {
		return nil, err
	}

	// apply setters
	doc, err := c.applyVirtualSetters(document)
	if err != nil {
		return nil, err
	} else if doc == nil {
		return nil, fmt.Errorf("no document")
	}

	document = *doc

	// add the id if it exists
	if id, ok := document["_id"]; ok {
		output["_id"] = id
	}

	for fieldName, field := range c.Fields {
		fieldPath := append(path, fieldName)
		fieldStr := strings.Join(fieldPath, ".")
		validated, err := field.walk(document[fieldName], fieldPath, options)
		if err != nil {
			return nil, err
		}
		if validated == nil {
			if field.Required {
				return nil, fmt.Errorf("required document path %q not set", fieldStr)
			}
			continue
		}

		// if it made it past the validations set the field
		output[fieldName] = validated
	}

	return &output, nil
}

// walk walks the schema field
func (c *SchemaField) walk(
	value interface{},
	path []string,
	options *walkOptions,
) (interface{}, error) {
	if c.isArray {
		return c.walkArray(value, path, options)
	}
	return c.walkSingle(value, path, options)
}

// walkArray walks a schema field that is an array
func (c *SchemaField) walkArray(
	value interface{},
	path []string,
	options *walkOptions,
) (interface{}, error) {
	pathStr := strings.Join(path, ".")

	// check required value
	if value == nil {
		if options.validateRequired && c.Required {
			return nil, fmt.Errorf("required document path %q not set", pathStr)
		}
		return nil, nil
	}

	// validate that value is an array
	if !helpers.IsArrayLike(value) {
		if options.validateTypes {
			return nil, fmt.Errorf("document path %q is not an array", pathStr)
		}
	}

	// iterate through each item in the array
	output := make([]interface{}, 0)
	el := helpers.GetElement(value)
	for i := 0; i < el.Len(); i++ {
		item, err := c.walkSingle(
			el.Index(i).Interface(),
			append(path, fmt.Sprintf("%d", i)),
			options,
		)
		if err != nil {
			return nil, err
		}
		if item != nil {
			output = append(output, item)
		}
	}

	return &output, nil
}

// walkSingle walks a single field
func (c *SchemaField) walkSingle(
	value interface{},
	path []string,
	options *walkOptions,
) (interface{}, error) {
	pathStr := strings.Join(path, ".")

	// create a return func
	var resultFunc = func(value interface{}, err error) (interface{}, error) {
		// if there is already an error, return it
		if err != nil {
			return nil, err
		}

		// if there is no value
		if value == nil {
			// check the required validator
			if options.validateRequired {
				return nil, fmt.Errorf("required document path %q not set", pathStr)
			}
			return nil, nil
		}

		// run custom validators if specified
		if options.validateCustom && c.Validate != nil {
			for _, validateFunc := range *c.Validate {
				if err := validateFunc(value); err != nil {
					return nil, err
				}
			}
		}
		return value, err
	}

	// apply default
	if value == nil && options.applyDefaults && c.Default != nil {
		value = c.Default
	}

	// check required and mixed
	if value == nil || c.elementType == MixedType {
		return resultFunc(value, nil)
	}

	// get kinds
	switch kind := helpers.GetKind(value); kind {

	// array values at this point should only be an objectid
	case reflect.Array, reflect.Slice:
		if c.elementType != ObjectIDType {
			if options.validateTypes {
				return resultFunc(nil, fmt.Errorf("document path %q is not a valid %s", pathStr, c.elementType))
			}
			return resultFunc(nil, nil)
		}
		return resultFunc(value, nil)

	// string can potentially be an object id
	case reflect.String:
		if c.elementType != StringType && c.elementType != ObjectIDType {
			if options.validateTypes {
				return resultFunc(nil, fmt.Errorf("document path %q is not a valid %s", pathStr, c.elementType))
			}
			return resultFunc(nil, nil)
		}
		if c.elementType == ObjectIDType && options.castObjectID {
			oid, err := primitive.ObjectIDFromHex(value.(string))
			if err != nil {
				if options.validateTypes {
					return resultFunc(nil, fmt.Errorf("document path %q failed to cast ObjectID", pathStr))
				}
				return resultFunc(nil, nil)
			}
			return resultFunc(oid, nil)
		}
		return resultFunc(value, nil)

	// bools are bools
	case reflect.Bool:
		if c.elementType != BoolType {
			if options.validateTypes {
				return resultFunc(nil, fmt.Errorf("document path %q is not a valid %s", pathStr, c.elementType))
			}
			return resultFunc(nil, nil)
		}
		return resultFunc(value, nil)

	// ints are ints
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if c.elementType != IntType {
			if options.validateTypes {
				return resultFunc(nil, fmt.Errorf("document path %q is not a valid %s", pathStr, c.elementType))
			}
			return resultFunc(nil, nil)
		}
		return resultFunc(value, nil)

	// floats are floats
	case reflect.Float32, reflect.Float64:
		if c.elementType != FloatType {
			if options.validateTypes {
				return resultFunc(nil, fmt.Errorf("document path %q is not a valid %s", pathStr, c.elementType))
			}
			return resultFunc(nil, nil)
		}
		return resultFunc(value, nil)

	// maps should be schema types
	case reflect.Map:
		schema := getSchema(c.elementType)
		if schema == nil {
			if options.validateTypes {
				return resultFunc(nil, fmt.Errorf("document path %q is not a valid schema", pathStr))
			}
			return resultFunc(nil, nil)
		}
		subDoc, err := schema.walk(value, path, options)
		return resultFunc(subDoc, err)
	}

	if options.validateTypes {
		return resultFunc(nil, fmt.Errorf("cannot determine data type at document path %q", pathStr))
	}
	return resultFunc(nil, nil)
}
