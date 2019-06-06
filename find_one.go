package gongo

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FindOne finds one record
func (c *Model) FindOne(
	filter *bson.M,
	target interface{},
	opts ...*options.FindOneOptions,
) error {
	return c.FindOneWithTimeout(filter, target, nil, opts...)
}

// FindOneWithTimeout record matching query
func (c *Model) FindOneWithTimeout(
	filter *bson.M,
	target interface{},
	timeout *int,
	opts ...*options.FindOneOptions,
) error {
	tgtVal := reflect.ValueOf(target)
	if tgtVal.Kind() != reflect.Ptr || tgtVal.IsNil() {
		return ErrInvalidTarget
	}

	ctx, cancelFunc := newContext(timeout)
	defer cancelFunc()

	// apply virtuals to the filter
	query, err := c.schema.applyVirtualQueryDocument(filter)
	if err != nil {
		return err
	}

	// perform the findOne operation
	result := c.Collection().FindOne(ctx, query, opts...)

	// check for errors in the result
	if err := result.Err(); err != nil {
		return err
	}

	// decode the result to a temp object
	temp := bson.M{}
	if err := result.Decode(&temp); err != nil {
		return err
	}

	// apply virtual getters
	if err := c.schema.applyVirtualGetters(temp); err != nil {
		return err
	}

	// finally convert the updated document to the target interface
	return toInterface(temp, target)
}
