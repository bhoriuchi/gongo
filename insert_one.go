package gongo

import (
	"fmt"
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InsertOne inserts one document
func (c *Model) InsertOne(
	doc bson.M,
	target interface{},
	opts ...*options.InsertOneOptions,
) error {
	return c.InsertOneWithTimeout(doc, target, nil, opts...)
}

// InsertOneWithTimeout inserts one document
func (c *Model) InsertOneWithTimeout(
	doc bson.M,
	target interface{},
	timeout *int,
	opts ...*options.InsertOneOptions,
) error {
	tgtVal := reflect.ValueOf(target)
	if tgtVal.Kind() != reflect.Ptr && !tgtVal.IsNil() {
		return ErrInvalidTarget
	}

	// apply virtuals
	document, err := c.schema.applyVirtualSetters(doc)
	if err != nil {
		return err
	}

	// create a context
	ctx, cancelFunc := newContext(timeout)
	defer cancelFunc()

	// insert the document
	result, err := c.Collection().InsertOne(ctx, document, opts...)
	if err != nil {
		return err
	}

	if result.InsertedID == nil {
		return fmt.Errorf("insert failed, no ObjectId returned")
	}

	// get the document and return it
	if !tgtVal.IsNil() {
		return c.FindByID(result.InsertedID, target)
	}
	return nil
}
