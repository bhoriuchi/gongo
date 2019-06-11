package gongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
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

// FindOneAndUpdate finds a document and updates it
func (c *Model) FindOneAndUpdate(
	filter interface{},
	update interface{},
	opts ...*options.FindOneAndUpdateOptions,
) (*Document, error) {
	return c.FindOneAndUpdateWithTimeout(filter, update, nil, opts...)
}

// FindOneAndUpdateWithTimeout finds a document and updates it
func (c *Model) FindOneAndUpdateWithTimeout(
	filter interface{},
	update interface{},
	timeout *int,
	opts ...*options.FindOneAndUpdateOptions,
) (*Document, error) {
	if update == nil {
		return nil, fmt.Errorf("no update specified")
	}
	m := bson.M{}
	if filter != nil {
		if err := weakDecode(filter, &m); err != nil {
			return nil, err
		}
	}

	// get a query
	query, err := c.applyVirtualQueryDocument(&m)
	if err != nil {
		return nil, err
	}

	// create a working document
	doc := bson.M{}
	if err := weakDecode(update, &doc); err != nil {
		return nil, err
	}

	document, err := c.schema.applyVirtualSetters(doc)
	if err != nil {
		return nil, err
	}

	// apply pre-middleware
	if err := c.schema.applyPreMiddleware("findOneAndUpdate", *document); err != nil {
		return nil, err
	}

	// filter undefined fields
	document = c.schema.filterUndefined(document)

	// validate the document
	if err := c.schema.validate(document, []string{}, true); err != nil {
		return nil, err
	}

	// create a context
	ctx, cancelFunc := newContext(timeout)
	defer cancelFunc()

	// perform the find operation
	result := c.Collection().FindOneAndUpdate(
		ctx,
		query,
		bson.M{"$set": document},
		opts...,
	)
	if err := result.Err(); err != nil {
		return nil, err
	}

	var temp bson.M
	if err := result.Decode(&temp); err != nil {
		return nil, err
	}

	// apply post middleware
	if err := c.schema.applyPostMiddleware("findOneAndUpdate", temp, nil); err != nil {
		return nil, err
	}

	// return a new document
	return c.New(temp)
}

// FindOneAndDelete finds a document and deletes it
func (c *Model) FindOneAndDelete(
	filter interface{},
	opts ...*options.FindOneAndDeleteOptions,
) (*Document, error) {
	return c.FindOneAndDeleteWithTimeout(filter, nil, opts...)
}

// FindOneAndDeleteWithTimeout finds a document and deletes it
func (c *Model) FindOneAndDeleteWithTimeout(
	filter interface{},
	timeout *int,
	opts ...*options.FindOneAndDeleteOptions,
) (*Document, error) {
	m := bson.M{}
	if filter != nil {
		if err := weakDecode(filter, &m); err != nil {
			return nil, err
		}
	}

	// get a query
	query, err := c.applyVirtualQueryDocument(&m)
	if err != nil {
		return nil, err
	}

	// apply pre-middleware
	if err := c.schema.applyPreMiddleware("findOneAndDelete", *query); err != nil {
		return nil, err
	}

	// create a context
	ctx, cancelFunc := newContext(timeout)
	defer cancelFunc()

	// perform the update
	result := c.Collection().FindOneAndDelete(ctx, query, opts...)
	if err := result.Err(); err != nil {
		return nil, err
	}

	var temp bson.M
	if err := result.Decode(&temp); err != nil {
		return nil, err
	}

	// apply post middleware
	if err := c.schema.applyPostMiddleware("findOneAndDelete", temp, nil); err != nil {
		return nil, err
	}

	// return a new document
	return c.New(temp)
}
