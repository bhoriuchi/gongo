package gongo

import (
	"reflect"

	"github.com/bhoriuchi/gongo/helpers"
	"go.mongodb.org/mongo-driver/bson"
)

// VirtualFieldMap a map of virtual types
type VirtualFieldMap map[string]*VirtualConfig

// VirtualGetFunc for resolving virtual
type VirtualGetFunc func(doc bson.M) (interface{}, error)

// VirtualSetFunc for resolving virtual
type VirtualSetFunc func(value interface{}, doc bson.M) error

// VirtualConfig defines the virtual config
type VirtualConfig struct {
	Name string
	Get  VirtualGetFunc
	Set  VirtualSetFunc
}

// Virtual adds a virtual field config
func (c *Schema) Virtual(config *VirtualConfig) {
	if config == nil || config.Name == "" {
		return
	}
	if c.virtuals == nil {
		c.virtuals = &VirtualFieldMap{}
	}
	virtuals := *c.virtuals
	if _, ok := virtuals[config.Name]; ok {
		return
	}
	virtuals[config.Name] = config
}

// apply virtual setters creates a new document by setting virtual fields
// and keeping the non-virtual fields
func (c *Schema) applyVirtualSetters(doc bson.M) (*bson.M, error) {
	virtuals := *c.virtuals
	newDoc := bson.M{}
	for k, v := range doc {
		if config, ok := virtuals[k]; ok {
			if err := config.Set(v, newDoc); err != nil {
				return nil, err
			}
		} else {
			newDoc[k] = v
		}
	}
	return &newDoc, nil
}

// apply virtuals getters
func (c *Schema) applyVirtualGetters(doc bson.M) error {
	for _, v := range *c.virtuals {
		value, err := v.Get(doc)
		if err != nil {
			return err
		}
		doc[v.Name] = value
	}
	return nil
}

// converts the filter document to a valid one by replacing string versions of object ids
// and pointing virtual values to the right keys
func (c *Model) applyVirtualQueryDocument(filter *bson.M) (*bson.M, error) {
	if filter == nil {
		return &bson.M{}, nil
	}
	query, err := c.deepQueryBuild(*filter)
	if err != nil {
		return nil, err
	}
	newFilter := query.(bson.M)
	return &newFilter, nil
}

// performs a deep build of the query
func (c *Model) deepQueryBuild(obj interface{}) (interface{}, error) {
	// check for object id and return right away
	if helpers.IsObjectID(obj) {
		return obj, nil
	}

	// look at each kind
	switch kind := helpers.GetKind(obj); kind {

	// handle slice/array
	case reflect.Slice, reflect.Array:
		s := reflect.ValueOf(obj)
		result := make([]interface{}, 0)
		for i := 0; i < s.Len(); i++ {
			value, err := c.deepQueryBuild(s.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil

	// handle maps
	case reflect.Map:
		result := bson.M{}
		virtuals := *c.schema.virtuals
		original := reflect.ValueOf(obj)
		for _, key := range original.MapKeys() {
			k := key.Interface().(string)
			v := original.MapIndex(key).Interface()

			// get the updated value by calling deep query build on it
			value, err := c.deepQueryBuild(v)
			if err != nil {
				return nil, err
			}

			// if the value is virtual, use the setter function
			// otherwise just set the value as is
			if config, ok := virtuals[k]; ok {
				if err := config.Set(value, result); err != nil {
					return nil, err
				}
			} else {
				result[k] = value
			}
		}
		return result, nil
	}

	return obj, nil
}
