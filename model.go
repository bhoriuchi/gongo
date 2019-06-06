package gongo

import (
	"context"
	"fmt"

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
		// setup unique indexes
		uniq := tm.get("unique")
		trueValue := true
		if uniq != nil && *uniq == "true" {
			actualName := getActualFieldName(tm)
			if actualName != nil {
				_, err := c.Collection().Indexes().CreateOne(context.Background(), mongo.IndexModel{
					Keys:    map[string]int{*actualName: 1},
					Options: &options.IndexOptions{Unique: &trueValue},
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
