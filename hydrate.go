package gongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

// Hydrate hydrates the model with data from the database
func (c *Model) Hydrate(filter *bson.M) error {
	return c.HydrateWithTimeout(filter, nil)
}

// HydrateWithTimeout hydrates the model with data from the database
func (c *Model) HydrateWithTimeout(filter *bson.M, timeout *int) error {
	if filter == nil {
		return fmt.Errorf("no filter provided")
	}

	ctx, cancelFunc := newContext(timeout)
	defer cancelFunc()

	// apply virtuals to the filter
	query, err := c.applyVirtualQueryDocument(filter)
	if err != nil {
		return err
	}

	// decode to the document
	return c.Collection().FindOne(ctx, query).Decode(c.document)
}
