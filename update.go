package gongo

import (
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InsertOne inserts one document
func (c *Model) InsertOne(doc interface{}, opts ...*options.InsertOneOptions) (*Document, error) {
	return c.InsertOneWithTimeout(doc, nil, opts...)
}

// InsertOneWithTimeout inserts one document
func (c *Model) InsertOneWithTimeout(doc interface{}, timeout *int, opts ...*options.InsertOneOptions) (*Document, error) {
	return c.CreateWithTimeout(doc, timeout)
}
