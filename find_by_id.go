package gongo

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FindByID finds item by id
func (c *Model) FindByID(
	id interface{},
	target interface{},
	opts ...*options.FindOneOptions,
) error {
	return c.FindByIDWithTimeout(id, target, nil, opts...)
}

// FindByIDWithTimeout finds item by id
func (c *Model) FindByIDWithTimeout(
	id interface{},
	target interface{},
	timeout *int,
	opts ...*options.FindOneOptions,
) error {
	tgtVal := reflect.ValueOf(target)
	if tgtVal.Kind() != reflect.Ptr || tgtVal.IsNil() {
		return ErrInvalidTarget
	}
	return c.FindOneWithTimeout(&bson.M{"_id": id}, target, timeout, opts...)
}
