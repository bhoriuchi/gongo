package gongo

import (
	"context"
	"fmt"
	"time"

	"github.com/gertd/go-pluralize"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// New creates a new gongo instance
func New(database string, options *options.ClientOptions) *Gongo {
	return &Gongo{
		database:    database,
		options:     options,
		models:      make(map[string]*Model),
		fieldTagDef: DefaultFieldTagDefinition(),
		client:      nil,
	}
}

// Gongo main interface
type Gongo struct {
	database    string
	options     *options.ClientOptions
	models      map[string]*Model
	fieldTagDef *FieldTagDefinition
	client      *mongo.Client
}

// Connect connects to mongodb
func (c *Gongo) Connect() error {
	if c.database == "" {
		return fmt.Errorf("no database specified")
	}

	// connect the client
	client, err := mongo.Connect(context.Background(), c.options)
	if err != nil {
		return err
	}
	c.client = client

	// create indexes for each model
	for _, model := range c.models {
		if err := model.createIndexes(); err != nil {
			return err
		}
	}

	return nil
}

// Model creates a new model and adds it to gongo
func (c *Gongo) Model(schema *Schema, opts ...*ModelOptions) (*Model, error) {
	if schema == nil {
		return nil, fmt.Errorf("no schema provided")
	}

	// create some default model options
	options := &ModelOptions{
		Name:          schema.typeName,
		DontPluralize: false,
	}

	// check for specified model options and update
	if len(opts) > 0 && opts[0] != nil {
		o := opts[0]
		if o.Name != "" {
			options.Name = o.Name
		}
		options.DontPluralize = o.DontPluralize
	}

	// format the collection name
	collectionName := toSnakeCase(options.Name)
	if !options.DontPluralize {
		collectionName = pluralize.Plural(collectionName)
	}

	// check for already registered models
	if _, ok := c.models[schema.typeName]; ok {
		return nil, fmt.Errorf("type %q has already been registered", schema.typeName)
	}

	// create the model
	// field tags are mapped at model registration because
	// the schema allows for the tag definition to be overriden
	model := &Model{
		schema:         schema,
		collectionName: collectionName,
		gongo:          c,
		fieldTagMap:    mapStructTags(schema.refType, schema.fieldTagDef.Tags()),
		baseModel:      nil,
		document:       nil,
	}

	c.models[schema.typeName] = model
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
