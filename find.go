package gongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Find finds documents
func (c *Model) Find(filter interface{}, opts ...*options.FindOptions) (DocumentList, error) {
	return c.FindWithTimeout(filter, nil, opts...)
}

// FindWithTimeout finds documents
func (c *Model) FindWithTimeout(filter interface{}, timeout *int, opts ...*options.FindOptions) (DocumentList, error) {
	results := make(DocumentList, 0)
	m := bson.M{}
	if filter != nil {
		if err := weakDecode(filter, &m); err != nil {
			return nil, err
		}
	}

	query, err := c.applyVirtualQueryDocument(&m)
	if err != nil {
		return nil, err
	}

	// create a context
	ctx, cancelFunc := newContext(timeout)
	defer cancelFunc()

	// perform the find operation
	cur, err := c.Collection().Find(ctx, query, opts...)
	if err != nil {
		return nil, err
	}

	defer cur.Close(ctx)
	if err := cur.Err(); err != nil {
		return nil, err
	}

	// decode all the results
	var temp []bson.M
	if err := cur.All(ctx, &temp); err != nil {
		return nil, err
	}

	for _, result := range temp {
		doc, err := c.New(result)
		if err != nil {
			return nil, err
		}
		results = append(results, doc)
	}

	return results, nil
}

// FindOne finds one document
func (c *Model) FindOne(filter interface{}, opts ...*options.FindOneOptions) (*Document, error) {
	return c.FindOneWithTimeout(filter, nil, opts...)
}

// FindOneWithTimeout finds one document
func (c *Model) FindOneWithTimeout(filter interface{}, timeout *int, opts ...*options.FindOneOptions) (*Document, error) {
	m := bson.M{}
	if filter != nil {
		if err := weakDecode(filter, &m); err != nil {
			return nil, err
		}
	}

	query, err := c.applyVirtualQueryDocument(&m)
	if err != nil {
		return nil, err
	}

	// create a context
	ctx, cancelFunc := newContext(timeout)
	defer cancelFunc()

	// perform the find operation
	result := c.Collection().FindOne(ctx, query, opts...)
	if err := result.Err(); err != nil {
		return nil, err
	}

	var temp bson.M
	if err := result.Decode(&temp); err != nil {
		return nil, err
	}

	return c.New(temp)
}

// FindByID finds one document by id
func (c *Model) FindByID(id interface{}, opts ...*options.FindOneOptions) (*Document, error) {
	return c.FindByIDWithTimeout(id, nil, opts...)
}

// FindByIDWithTimeout finds one document by id
func (c *Model) FindByIDWithTimeout(id interface{}, timeout *int, opts ...*options.FindOneOptions) (*Document, error) {
	if id == nil {
		return nil, fmt.Errorf("required id not provided")
	}
	return c.FindOneWithTimeout(bson.M{"_id": id}, timeout, opts...)
}
