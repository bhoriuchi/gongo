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
	connected   bool
	database    string
	options     *options.ClientOptions
	models      map[string]*Model
	validators  map[string]ValidateFunc
	fieldTagDef *FieldTagDefinition
	client      *mongo.Client
	log         Log
	hub         chan string
}

// New creates a new gongo instance
func New(database string, options *options.ClientOptions) *Gongo {
	g := &Gongo{
		connected:   false,
		database:    database,
		options:     options,
		models:      make(map[string]*Model),
		validators:  make(map[string]ValidateFunc),
		fieldTagDef: DefaultFieldTagDefinition(),
		client:      nil,
		log:         Log{},
		hub:         make(chan string),
	}

	// perform tasks on channel messages
	go func() {
		switch msg := <-g.hub; msg {
		case "connected":
			g.log.Debug("connected to mongodb")
			for _, model := range g.models {
				if !model.createdIndexes {
					g.log.Debugf("creating indexes for %s", model.collectionName)
					if err := model.createIndexes(); err != nil {
						g.log.Errorf("CreateIndex on %q error: %s", model.collectionName, err.Error())
					}
				}
			}
		}
	}()

	return g
}

// WithLogger sets a logger
func (c *Gongo) WithLogger(logger Logger) *Gongo {
	c.log = Log{logger: logger}
	return c
}

// WithTagDefinition adds a custom tag definition
func (c *Gongo) WithTagDefinition(definition *FieldTagDefinition) *Gongo {
	if definition == nil {
		c.log.Error("WithTagDefinition: no tag definition provided")
		return c
	} else if err := definition.Validate(); err != nil {
		c.log.Errorf("WithTagDefinition: %s", err.Error())
		return c
	}
	c.fieldTagDef = definition
	return c
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

	// validate the field tag definition
	if err := c.fieldTagDef.Validate(); err != nil {
		return err
	}

	c.connected = true
	c.hub <- "connected"
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
	collectionName := options.Name
	if !options.DontSnakeCase {
		collectionName = helpers.ToSnakeCase(collectionName)
	}
	if !options.DontPluralize {
		collectionName = pluralize.Plural(collectionName)
	}

	// check for already registered models
	if _, ok := c.models[schema.typeName]; ok {
		return nil, fmt.Errorf("type %q has already been registered", schema.typeName)
	}

	// create field tag map
	fieldTagMap := mapStructTags(schema.ref, c.fieldTagDef.Tags())

	// create the model
	// field tags are mapped at model registration because
	// the schema allows for the tag definition to be overriden
	model := &Model{
		schema:         schema,
		collectionName: collectionName,
		createdIndexes: false,
		gongo:          c,
		fieldTagMap:    fieldTagMap,
		baseModel:      nil,
		document:       nil,
	}

	c.models[schema.typeName] = model

	// create indexes if connected
	if c.connected && !model.createdIndexes {
		c.log.Debugf("creating indexes for %s", model.collectionName)
		if err := model.createIndexes(); err != nil {
			c.log.Errorf("CreateIndex on %q error: %s", model.collectionName, err.Error())
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
