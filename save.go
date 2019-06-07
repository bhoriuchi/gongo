package gongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

// Save saves a doument
func (c *Model) Save() error {
	return c.SaveWithTimeout(nil)
}

// SaveWithTimeout saves the document
func (c *Model) SaveWithTimeout(timeout *int) error {
	var opError error

	// apply virtuals
	document, err := c.schema.applyVirtualSetters(*c.document)
	if err != nil {
		return err
	}
	c.document = document

	// validate the document
	if err := c.Validate(); err != nil {
		return err
	}

	// apply pre-middleware
	if err := c.schema.applyPreMiddleware("save", *c.document, c); err != nil {
		return err
	}

	// create a context
	ctx, cancelFunc := newContext(timeout)
	defer cancelFunc()

	// check if updating
	d := *c.document
	if id, ok := d["_id"]; ok {
		update := bson.M{}
		for k, v := range d {
			if k != "_id" {
				update[k] = v
			}
		}
		result, err := c.Collection().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
		if err != nil {
			opError = err
		} else if result.MatchedCount < 1 {
			opError = fmt.Errorf("failed to update %s", id)
		}

	} else {
		// insert the document
		result, err := c.Collection().InsertOne(ctx, document)
		if err != nil {
			opError = err
		} else {
			if result.InsertedID == nil {
				opError = fmt.Errorf("insert failed, no ObjectId returned")
			} else {
				d["_id"] = result.InsertedID
			}
		}
	}

	// apply post middleware
	if err := c.schema.applyPostMiddleware("save", d, c, opError); err != nil {
		opError = err
	}

	return opError
}
