package gongo

import (
	"context"

	"github.com/mitchellh/mapstructure"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ModelOptions options for the model
type ModelOptions struct {
	DontPluralize bool
	DontSnakeCase bool
}

// Model model
type Model struct {
	initialized    bool
	schema         *Schema
	gongo          *Gongo
	collectionName string
}

// Database returns the database object
func (c *Model) Database() *mongo.Database {
	return c.gongo.client.Database(c.gongo.database)
}

// Collection returns the collection object
func (c *Model) Collection() *mongo.Collection {
	return c.Database().Collection(c.collectionName)
}

// New creates a new instance of a model
func (c *Model) New(document interface{}) (*Document, error) {
	if document == nil {
		document = &bson.M{}
	}

	newDocument := Document{model: c}
	if err := newDocument.load(document, c.schema); err != nil {
		return nil, err
	}

	return &newDocument, nil
}

// Hydrate hydrates a model
func (c *Model) Hydrate(filter interface{}) (*Document, error) {
	return c.HydrateWithTimeout(filter, nil)
}

// HydrateWithTimeout hydrates a model
func (c *Model) HydrateWithTimeout(filter interface{}, timeout *int) (*Document, error) {
	q := bson.M{}
	if filter != nil {
		if err := mapstructure.WeakDecode(filter, &q); err != nil {
			return nil, err
		}
	}

	ctx, cancelFunc := newContext(timeout)
	defer cancelFunc()

	// apply virtuals to the filter
	query, err := c.applyVirtualQueryDocument(&q)
	if err != nil {
		return nil, err
	}

	// look for the result
	result := c.Collection().FindOne(ctx, query)
	if err := result.Err(); err != nil {
		return nil, err
	}

	// hydrate a temp
	temp := bson.M{}
	if err := result.Decode(&temp); err != nil {
		return nil, err
	}

	// load the data
	return c.New(temp)
}

// creates indexes
func (c *Model) createIndexes() error {
	uniq := true
	for name, field := range c.schema.Fields {
		if field.Unique {
			_, err := c.Collection().Indexes().CreateOne(context.Background(), mongo.IndexModel{
				Keys:    map[string]int{name: 1},
				Options: &options.IndexOptions{Unique: &uniq},
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
