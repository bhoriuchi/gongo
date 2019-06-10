package gongo

import (
	"fmt"
	"strings"

	"github.com/bhoriuchi/gongo/helpers"
	"github.com/mitchellh/mapstructure"
	"github.com/mitchellh/pointerstructure"
	"go.mongodb.org/mongo-driver/bson"
)

// Document a mongodb document wrapper
type Document struct {
	id    interface{}
	model *Model
	prev  *bson.M
	cur   *bson.M
	next  *bson.M
}

// ID returns the object id
func (c *Document) ID() interface{} {
	return c.id
}

// Get gets the current proposed change path
func (c *Document) Get(path string) (interface{}, error) {
	p := helpers.DotPathToSlashPath(path)
	return pointerstructure.Get(c.next, p)
}

// Set sets the current proposed change path
func (c *Document) Set(path string, value interface{}) error {
	p := helpers.DotPathToSlashPath(path)
	fieldPath := strings.Split(path, ".")
	if !c.model.schema.hasFieldPath(fieldPath) {
		return fmt.Errorf("undefined path %q cannot be set", path)
	}

	if _, err := pointerstructure.Set(c.next, p, value); err != nil {
		return err
	}

	// validate the changes, if they fail revert to the current document
	if err := c.Validate(); err != nil {
		c.revertCurrent()
		return err
	}

	return nil
}

// loads document data
func (c *Document) load(document interface{}, schema *Schema) error {
	// first conver to a map
	doc := bson.M{}
	prev := bson.M{}
	cur := bson.M{}
	next := bson.M{}

	// convert to bson
	if document != nil {
		if err := mapstructure.WeakDecode(document, &doc); err != nil {
			return err
		}
	}

	setDoc, err := schema.applyVirtualSetters(doc)
	if err != nil {
		return err
	}

	// filter non-schema fields and _id
	for k, v := range *setDoc {
		if _, ok := schema.Fields[k]; ok || k == "_id" {
			if k == "_id" {
				c.id = v
			} else {
				cur[k] = v
			}
		}
	}

	// copy to prev and next
	if err := mapstructure.WeakDecode(cur, &prev); err != nil {
		return err
	}
	if err := mapstructure.WeakDecode(cur, &next); err != nil {
		return err
	}

	c.prev = &prev
	c.cur = &cur
	c.next = &next

	return nil
}

// moves current to prev, next to cur, and leaves next alone
func (c *Document) moveNext() error {
	if err := mapstructure.WeakDecode(c.cur, c.prev); err != nil {
		return err
	}
	if err := mapstructure.WeakDecode(c.next, c.cur); err != nil {
		return err
	}
	return nil
}

// reverts the next to the current essentially removing any updates on the model
// this does not save the revert
func (c *Document) revertCurrent() error {
	next := bson.M{}
	if err := mapstructure.WeakDecode(c.cur, &next); err != nil {
		return err
	}
	c.next = &next
	return nil
}

// reverts the model to the previous version of the data
// this does not dave the revert
func (c *Document) revertPrevious() error {
	original := c.cur
	cur := bson.M{}
	if err := mapstructure.WeakDecode(c.prev, &cur); err != nil {
		return err
	}
	if err := c.revertCurrent(); err != nil {
		c.cur = original
		return err
	}
	c.cur = &cur
	return nil
}

// Rollback rolls back changes to previous
func (c *Document) Rollback() error {
	return c.RollbackWithTimeout(nil)
}

// RollbackWithTimeout reverts the model to the previous and saves
// if this operation fails, the state before the rollback is returned
func (c *Document) RollbackWithTimeout(timeout *int) error {
	prev := c.prev
	cur := c.cur
	next := c.next

	if err := c.revertPrevious(); err != nil {
		c.prev = prev
		c.cur = cur
		c.next = next
		return err
	}

	if err := c.SaveWithTimeout(timeout); err != nil {
		c.prev = prev
		c.cur = cur
		c.next = next
		return err
	}

	return nil
}

// Decode decodes the document to an interface
func (c *Document) Decode(target interface{}) error {
	if target == nil {
		return fmt.Errorf("no decode target provided")
	}

	// make a working copy
	doc := bson.M{}
	if err := mapstructure.WeakDecode(c.cur, &doc); err != nil {
		return err
	}

	// add the id to the document
	if c.id != nil {
		doc["_id"] = c.id
	}

	// apply getters
	if err := c.model.schema.applyVirtualGetters(doc); err != nil {
		return err
	}

	// decode the structure
	return mapstructure.WeakDecode(doc, target)
}

// Save saves a document
func (c *Document) Save() error {
	return c.SaveWithTimeout(nil)
}

// SaveWithTimeout saves a document
func (c *Document) SaveWithTimeout(timeout *int) error {
	// re-usable error handler
	errorFunc := func(doc bson.M, err error) error {
		// apply post middleware
		if err := c.model.schema.applyPostMiddleware("save", doc, err); err != nil {
			return err
		}
		return err
	}

	// first create a working document
	doc := bson.M{}
	if err := mapstructure.WeakDecode(c.next, &doc); err != nil {
		return err
	}

	// apply pre-middleware
	if err := c.model.schema.applyPreMiddleware("save", doc); err != nil {
		return err
	}

	// filter undefined fields
	doc = c.model.schema.copyInternalDocument(doc)

	// apply defaults
	c.model.schema.setDefaults(doc)

	// validate the document
	if err := c.model.schema.validate(&doc, []string{}); err != nil {
		return err
	}

	// create a context
	ctx, cancelFunc := newContext(timeout)
	defer cancelFunc()

	// save
	if c.id != nil {
		result, err := c.model.Collection().UpdateOne(
			ctx,
			bson.M{"_id": c.id},
			bson.M{"$set": doc},
		)
		if err != nil {
			return errorFunc(nil, err)
		} else if result.MatchedCount < 1 {
			return errorFunc(nil, fmt.Errorf("failed to update %s", c.id))
		}
	} else {
		result, err := c.model.Collection().InsertOne(
			ctx,
			doc,
		)
		if err != nil {
			return errorFunc(nil, err)
		} else if result.InsertedID == nil {
			return errorFunc(nil, fmt.Errorf("insert failed, no ObjectId returned"))
		}
		c.id = result.InsertedID
	}

	// apply post middleware
	err := c.model.schema.applyPostMiddleware("save", doc, nil)
	if err != nil {
		return err
	}

	// update the internal model with a filtered copy
	nextDoc := c.model.schema.copyInternalDocument(doc)
	c.next = &nextDoc
	return c.moveNext()
}
