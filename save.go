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
			return err
		} else if result.MatchedCount < 1 {
			return fmt.Errorf("failed to update %s", id)
		}

	} else {
		// insert the document
		result, err := c.Collection().InsertOne(ctx, document)
		if err != nil {
			return err
		}

		if result.InsertedID == nil {
			return fmt.Errorf("insert failed, no ObjectId returned")
		}
		d["_id"] = result.InsertedID
	}
	return nil
}
