package gongo

import (
	"context"
	"fmt"
	"reflect"

	"github.com/bhoriuchi/gongo/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ModelOptions options for the model
type ModelOptions struct {
	Name          string
	DontPluralize bool
}

// Model creates a model from the struct
type Model struct {
	schema         *Schema
	collectionName string
	gongo          *Gongo
	fieldTagMap    *fieldTagMap
	baseModel      *Model
	document       *bson.M
}

// New creates a new instance of a model
func (c *Model) New(document bson.M) *Model {
	// reference the base model
	baseModel := c
	if c.baseModel != nil {
		baseModel = c.baseModel
	}

	// create a new model
	model := Model{
		schema:         c.schema,
		collectionName: c.collectionName,
		gongo:          c.gongo,
		baseModel:      baseModel,
		document:       &document,
	}

	return &model
}

// Database returns the database object
func (c *Model) Database() *mongo.Database {
	return c.gongo.client.Database(c.gongo.database)
}

// Collection returns the collection object
func (c *Model) Collection() *mongo.Collection {
	return c.Database().Collection(c.collectionName)
}

// checks if the current model is the base model
// this is useful because the base model should not
// be used to perform database operations
// it should only be used as a reference
func (c *Model) isBaseModel() bool {
	return c.baseModel == nil
}

// creates indexes
func (c *Model) createIndexes() error {
	if !c.isBaseModel() {
		return fmt.Errorf("cannot call createIndexes on non-base model")
	}
	fm := *c.fieldTagMap
	for _, tm := range fm {
		fieldName, hasName := tm.getString(c.schema.fieldTagDef.Get("name"))
		uniqKeys, isUniq := tm.getList(c.schema.fieldTagDef.Get("unique"))

		// setup unique index
		if isUniq && len(uniqKeys) > 0 {
			// create a key map
			keyMap := make(map[string]int, 0)
			for _, name := range uniqKeys {
				// if the field list contains "true" use the field name as the
				// unique key if it exists and break, otherwise use the name in the list
				// lists can be used to create compound unique indexes
				if name == "true" && len(uniqKeys) == 1 {
					if hasName {
						keyMap[fieldName] = 1
					}
					break
				} else {
					keyMap[name] = 1
				}
			}

			// if there were keys
			if len(keyMap) > 0 {
				_, err := c.Collection().Indexes().CreateOne(context.Background(), mongo.IndexModel{
					Keys:    keyMap,
					Options: &options.IndexOptions{Unique: &isUniq},
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
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
	}

	return obj, nil
}
