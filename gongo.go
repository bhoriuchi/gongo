package gongo

import (
	"context"
	"fmt"
	"time"

	"github.com/bhoriuchi/gongo/helpers"
	"github.com/gertd/go-pluralize"
	"github.com/mitchellh/mapstructure"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

// Options gongo options
type Options struct {
	FieldTag        string
	DefaultDatabase string
}

// Gongo main interface
type Gongo struct {
	connected     bool
	clientOptions *options.ClientOptions
	options       *Options
	database      string
	models        map[string]*Model
	client        *mongo.Client
	hub           chan string
	fieldTag      string
}

// M gets a model from the registered models
func (c *Gongo) M(name string) *Model {
	if model, ok := c.models[name]; ok {
		return model
	}
	return nil
}

// New creates a new gongo instance
func New(opts ...*Options) *Gongo {
	g := &Gongo{
		connected: false,
		models:    make(map[string]*Model),
		hub:       make(chan string),
		options: &Options{
			FieldTag:        "json",
			DefaultDatabase: "test",
		},
	}

	// get options
	if len(opts) > 0 {
		o := opts[0]
		if o != nil {
			if o.FieldTag != "" {
				g.options.FieldTag = o.FieldTag
			}
			if o.DefaultDatabase != "" {
				g.options.DefaultDatabase = o.DefaultDatabase
			}
		}
	}

	return g
}

// Connect connects to mongodb, this should be performed
// after all schema and model setup has taken place
func (c *Gongo) Connect(connectionString string) error {
	if c.connected {
		return fmt.Errorf("already connected")
	}

	// build client options from connectionString and validate
	c.clientOptions = options.Client().ApplyURI(connectionString)
	if err := c.clientOptions.Validate(); err != nil {
		return err
	}

	// get database name from connection string or use the default
	cs, _ := connstring.Parse(connectionString)
	if cs.Database != "" {
		c.database = cs.Database
	} else {
		c.database = c.options.DefaultDatabase
	}

	// connect the client
	client, err := mongo.Connect(context.Background(), c.clientOptions)
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

// performs a weakDecode
func (c *Gongo) weakDecode(input, output interface{}) error {
	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		TagName:          c.options.FieldTag,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

// creates a new context
func newContext(timeout ...*int) (context.Context, context.CancelFunc) {
	if len(timeout) > 0 {
		to := timeout[0]
		if to != nil && *to > 0 {
			return context.WithTimeout(
				context.Background(),
				time.Duration(*to)*time.Second,
			)
		}
	}
	return context.Background(), func() {}
}
