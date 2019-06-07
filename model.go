package gongo

import (
	"context"
	"fmt"

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
	createdIndexes bool
	gongo          *Gongo
	fieldTagMap    *fieldTagMap
	baseModel      *Model
	document       *bson.M
}

// New creates a new instance of a model
func (c *Model) New(document *bson.M) *Model {
	if document == nil {
		document = &bson.M{}
	}

	// reference the base model
	baseModel := c
	if c.baseModel != nil {
		baseModel = c.baseModel
	}

	// create a new model
	model := Model{
		schema:         c.schema,
		collectionName: c.collectionName,
		createdIndexes: baseModel.createdIndexes,
		gongo:          c.gongo,
		fieldTagMap:    c.fieldTagMap,
		baseModel:      baseModel,
		document:       document,
	}

	return &model
}

// Set sets a field on the document
func (c *Model) Set(fieldName string, value interface{}) *Model {
	if c.document != nil && fieldName != "" {
		doc := *c.document
		doc[fieldName] = value
	}
	return c
}

// Get gets the field value from the document
func (c *Model) Get(fieldName string) interface{} {
	if c.document != nil && fieldName != "" {
		doc := *c.document
		if val, ok := doc[fieldName]; ok {
			return val
		}
	}
	return nil
}

// Database returns the database object
func (c *Model) Database() *mongo.Database {
	return c.gongo.client.Database(c.gongo.database)
}

// Collection returns the collection object
func (c *Model) Collection() *mongo.Collection {
	return c.Database().Collection(c.collectionName)
}

// Decode decodes the document to the target
func (c *Model) Decode(target interface{}) error {
	// create a copy of the internal document to modify
	// with the virtual getters
	newDoc := c.copyDocument()
	if err := c.schema.applyVirtualGetters(*newDoc); err != nil {
		return err
	}

	// finally decode to the provided interface
	return helpers.ToInterface(newDoc, target)
}

// checks if the current model is the base model
// this is useful because the base model should not
// be used to perform database operations
// it should only be used as a reference
func (c *Model) isBaseModel() bool {
	return c.baseModel == nil
}

// copyDocument creates a copy of the document
func (c *Model) copyDocument() *bson.M {
	newDoc := bson.M{}
	if c.document != nil {
		for k, v := range *c.document {
			newDoc[k] = v
		}
	}
	return &newDoc
}

// creates indexes
func (c *Model) createIndexes() error {
	if !c.isBaseModel() {
		return fmt.Errorf("cannot call createIndexes on non-base model")
	}
	fm := *c.fieldTagMap
	for _, tm := range fm {
		fieldName, hasName := tm.getString(c.gongo.fieldTagDef.Get("name"))
		uniqKeys, isUniq := tm.getList(c.gongo.fieldTagDef.Get("unique"))

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
