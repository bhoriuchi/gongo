package gongo

import (
	"fmt"
	"strings"

	"github.com/bhoriuchi/gongo/helpers"
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

// DocumentList a list of documents
type DocumentList []*Document

// Decode decodes all of the documents in a document list to a target
func (c *DocumentList) Decode(target interface{}) error {
	l := *c
	var g *Gongo

	list := make([]interface{}, len(l))
	for i, doc := range l {
		if i == 0 {
			g = doc.model.gongo
		}
		v := bson.M{}
		if err := doc.Decode(&v); err != nil {
			return err
		}
		list[i] = &v
	}
	if g == nil {
		return helpers.ToInterface(list, target)
	}
	return g.weakDecode(list, target)
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
	next := bson.M{}

	// convert to bson
	if document != nil {
		if err := c.model.gongo.weakDecode(document, &doc); err != nil {
			return err
		}
	}

	setDoc, err := schema.applyVirtualSetters(doc)
	if err != nil {
		return err
	}

	// filter non-schema fields and _id
	cur := schema.filterUndefined(setDoc, c.model)
	if cur != nil {
		m := *cur
		if id, ok := m["_id"]; ok {
			c.id = id
		}
	}

	// copy to prev and next
	if err := c.model.gongo.weakDecode(cur, &prev); err != nil {
		return err
	}
	if err := c.model.gongo.weakDecode(cur, &next); err != nil {
		return err
	}

	c.prev = &prev
	c.cur = cur
	c.next = &next

	return nil
}

// moves current to prev, next to cur, and leaves next alone
func (c *Document) moveNext() error {
	if err := c.model.gongo.weakDecode(c.cur, c.prev); err != nil {
		return err
	}
	if err := c.model.gongo.weakDecode(c.next, c.cur); err != nil {
		return err
	}
	return nil
}

// reverts the next to the current essentially removing any updates on the model
// this does not save the revert
func (c *Document) revertCurrent() error {
	next := bson.M{}
	if err := c.model.gongo.weakDecode(c.cur, &next); err != nil {
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
	if err := c.model.gongo.weakDecode(c.prev, &cur); err != nil {
		return err
	}
	if err := c.revertCurrent(); err != nil {
		c.cur = original
		return err
	}
	c.cur = &cur
	return nil
}

// Decode decodes the document to an interface
func (c *Document) Decode(target interface{}) error {
	if target == nil {
		return fmt.Errorf("no decode target provided")
	}

	// make a working copy
	doc := bson.M{}
	if err := c.model.gongo.weakDecode(c.cur, &doc); err != nil {
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
	return c.model.gongo.weakDecode(doc, target)
}

// Save saves a document
func (c *Document) Save(timeout ...*int) error {
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
	if err := c.model.gongo.weakDecode(c.next, &doc); err != nil {
		return err
	}

	// apply pre-middleware
	if err := c.model.schema.applyPreMiddleware("save", doc); err != nil {
		return err
	}

	// filter undefined fields
	document := c.model.schema.filterUndefined(&doc, c.model)

	// apply defaults
	c.model.schema.setDefaults(*document)

	// validate the document
	if err := c.model.schema.validate(document, []string{}, false, c.model); err != nil {
		return err
	}

	// create a context
	ctx, cancelFunc := newContext(timeout...)
	defer cancelFunc()

	// save
	if c.id != nil {
		result, err := c.model.Collection().UpdateOne(
			ctx,
			bson.M{"_id": c.id},
			bson.M{"$set": document},
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
	nextDoc := c.model.schema.copyInternalDocument(*document)
	c.next = &nextDoc
	return c.moveNext()
}
