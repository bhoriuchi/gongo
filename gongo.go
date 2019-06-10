package gongo

import (
	"context"
	"fmt"
	"time"

	"github.com/bhoriuchi/gongo/helpers"
	"github.com/gertd/go-pluralize"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Gongo main interface
type Gongo struct {
	connected bool
	database  string
	options   *options.ClientOptions
	models    map[string]*Model
	client    *mongo.Client
	hub       chan string
}

// New creates a new gongo instance
func New(database string, options *options.ClientOptions) *Gongo {
	g := &Gongo{
		connected: false,
		database:  database,
		options:   options,
		models:    make(map[string]*Model),
		client:    nil,
		hub:       make(chan string),
	}

	return g
}

// Connect connects to mongodb, this should be performed
// after all schema and model setup has taken place
func (c *Gongo) Connect() error {
	if c.database == "" {
		return fmt.Errorf("no database specified")
	} else if c.connected {
		return fmt.Errorf("already connected")
	}

	// connect the client
	client, err := mongo.Connect(context.Background(), c.options)
	if err != nil {
		return err
	}
	c.client = client
	c.connected = true

	// emit an event to the hub
	go func() { c.hub <- "connected" }()

	// create indexes
	for _, model := range c.models {
		if !model.initialized {
			if err := model.createIndexes(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Model creates a new model and adds it to gongo
func (c *Gongo) Model(name string, schema *Schema, opts ...*ModelOptions) (*Model, error) {
	if schema == nil {
		return nil, fmt.Errorf("no schema provided")
	} else if name == "" {
		return nil, fmt.Errorf("no model name specified")
	}

	// initialize the schema
	if err := schema.init(); err != nil {
		return nil, err
	}

	// create some default model options
	options := &ModelOptions{
		DontPluralize: false,
		DontSnakeCase: false,
	}

	// check for specified model options and update
	if len(opts) > 0 && opts[0] != nil {
		o := opts[0]
		options.DontPluralize = o.DontPluralize
		options.DontSnakeCase = o.DontSnakeCase
	}

	// format the collection name
	collectionName := name
	if !options.DontSnakeCase {
		collectionName = helpers.ToSnakeCase(collectionName)
	}
	if !options.DontPluralize {
		collectionName = pluralize.Plural(collectionName)
	}

	// check for already registered models
	if _, ok := c.models[name]; ok {
		return nil, fmt.Errorf("type %q has already been registered", name)
	}

	// create a copy of the schema
	newSchema := schema.copy()

	// add id virtual by default
	if newSchema.Options == nil || newSchema.Options.ID == nil || *newSchema.Options.ID != false {
		newSchema.Virtual(&VirtualConfig{
			Name: "id",
			Get:  helpers.VirtualGetObjectIDAsHexString("_id"),
			Set:  helpers.VirtualSetObjectID("_id"),
		})
	}

	// create the model
	// field tags are mapped at model registration because
	// the schema allows for the tag definition to be overriden
	model := &Model{
		initialized:    false,
		schema:         newSchema,
		gongo:          c,
		collectionName: collectionName,
	}
	c.models[name] = model

	// if connected already, build the indexes
	if c.connected && !model.initialized {
		model.initialized = true
		if err := model.createIndexes(); err != nil {
			return model, err
		}
	}

	return model, nil
}

// creates a new context
func newContext(timeout *int) (context.Context, context.CancelFunc) {
	if timeout != nil && *timeout > 0 {
		return context.WithTimeout(
			context.Background(),
			time.Duration(*timeout)*time.Second,
		)
	}
	return context.Background(), func() {}
}
