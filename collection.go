package gongo

/*
import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Collection provides more concise mongodb operations
type Collection struct {
	collection     *mongo.Collection
	queryTimeout   time.Duration
	objectIDFields *[]string
	virtualFields  *map[string]string
	fieldMap       *FieldTagMap
}

// CollectionOptions options for the collection
type CollectionOptions struct {
	QueryTimeout   time.Duration
	ObjectIDFields *[]string
	VirtualFields  *map[string]string
}

// NewCollection creates a new collection wrapper
func NewCollection(collection *mongo.Collection, t interface{}, opts ...*CollectionOptions) *Collection {
	fieldMap := MapStructTags(t, fieldTags)
	fm := *fieldMap

	coll := Collection{
		collection: collection,
		fieldMap:   fieldMap,
	}

	if len(opts) > 0 {
		coll.queryTimeout = opts[0].QueryTimeout
		coll.objectIDFields = opts[0].ObjectIDFields
		coll.virtualFields = opts[0].VirtualFields
	}
	if coll.queryTimeout < 1 {
		coll.queryTimeout = 5
	}

	// get the fields tags with mongo_type:"ObjectId"
	if coll.objectIDFields == nil {
		idFields := make([]string, 0)
		for _, tm := range fm {
			mongoType := tm.Get("mongo_type")
			if mongoType != nil && *mongoType == "ObjectId" {
				actualName := getActualFieldName(tm)
				if actualName != nil {
					idFields = append(idFields, *actualName)
				}
			}
		}

		coll.objectIDFields = &idFields
	}

	// get the virtual fields and their maps
	if coll.virtualFields == nil {
		virtualFields := make(map[string]string)
		for _, tm := range fm {
			virtual := tm.Get("mongo_virtual")
			if virtual != nil && *virtual != "" {
				fieldName := tm.Get("name")
				if fieldName != nil && *fieldName != "" {
					virtualFields[*fieldName] = *virtual
				}
			}
		}

		coll.virtualFields = &virtualFields
	}

	// set up unique indexes
	for _, tm := range fm {
		uniq := tm.Get("unique")
		trueValue := true
		if uniq != nil && *uniq == "true" {
			actualName := getActualFieldName(tm)
			if actualName != nil {
				collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
					Keys:    map[string]int{*actualName: 1},
					Options: &options.IndexOptions{Unique: &trueValue},
				})
			}
		}
	}

	return &coll
}

// Collection returns the collection object
func (c *Collection) Collection() *mongo.Collection {
	return c.collection
}

// CreateUniqueIndex creates a unique index
func (c *Collection) CreateUniqueIndex(field string) {
	uniq := true
	c.collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    map[string]int{field: 1},
		Options: &options.IndexOptions{Unique: &uniq},
	})
}

// castIDFields casts string field values with the
// collection specified id field names to objectid types
func (c *Collection) sanitizeObject(doc, target interface{}, operation string) error {
	tgtVal := reflect.ValueOf(target)
	if tgtVal.Kind() != reflect.Ptr || tgtVal.IsNil() {
		return ErrInvalidTarget
	}

	if doc == nil {
		return nil
	}

	// perform a deep map modify on the keys and values
	result, err := DeepMapModify(
		doc,
		func(value, key interface{}) (interface{}, error) { // value func
			// get the mongo type
			mongoType := c.fieldMap.Get(key.(string), "mongo_type")

			// handle object ids
			if mongoType != nil && *mongoType == "ObjectId" {
				idObj, err := primitive.ObjectIDFromHex(fmt.Sprintf("%v", value))
				if err != nil {
					return nil, err
				}
				return idObj, nil
			}

			return value, nil
		},
		func(value interface{}, key interface{}) (interface{}, error) { // key func
			tm := c.fieldMap.Field(key.(string))
			if tm == nil {
				return key, nil
			}

			mongoVirtual := tm.Get("mongo_virtual")
			if mongoVirtual != nil && *mongoVirtual != "" {
				virtual := *mongoVirtual
				if virtual[:1] == "$" {
					return virtual[1:], nil
				}
			}
			return key, nil
		},
	)

	if err != nil {
		return err
	}

	// set the modified object to the original object
	reflect.ValueOf(target).Elem().Set(reflect.ValueOf(&result).Elem())
	return nil
}

// applies the virtual keys
func (c *Collection) applyVirtuals(doc interface{}, target interface{}) error {
	tgtVal := reflect.ValueOf(target)
	if tgtVal.Kind() != reflect.Ptr || tgtVal.IsNil() {
		return ErrInvalidTarget
	}

	if doc == nil {
		return nil
	}

	// perform a deep map modify on the keys and values
	result, err := DeepMapModify(
		doc,
		func(value interface{}, key interface{}) (interface{}, error) { // value func
			return value, nil
		},
		func(value interface{}, key interface{}) (interface{}, error) { // key func
			vKey := fmt.Sprintf("$%v", key)
			tagMap := c.fieldMap.FindOneByName("mongo_virtual", vKey)
			if tagMap != nil {
				fieldName := tagMap.Get("name")
				if fieldName != nil && *fieldName != "" {
					return *fieldName, nil
				}
			}

			return key, nil
		},
	)

	if err != nil {
		return err
	}

	return ToInterface(result, target)
}

// FindByID finds item by id
func (c *Collection) FindByID(id interface{}, target interface{}) error {
	tgtVal := reflect.ValueOf(target)
	if tgtVal.Kind() != reflect.Ptr || tgtVal.IsNil() {
		return ErrInvalidTarget
	}
	return c.FindOne(bson.M{"_id": id}, target)
}

// Find records matching query
func (c *Collection) Find(filter interface{}, target interface{}, opts ...*options.FindOptions) error {
	tgtVal := reflect.ValueOf(target)
	if tgtVal.Kind() != reflect.Ptr || tgtVal.IsNil() {
		return ErrInvalidTarget
	}

	// create a context
	ctx, cancelFunc := context.WithTimeout(context.Background(), c.queryTimeout*time.Second)
	defer cancelFunc()

	// sanitize the filter
	var sanitizedFilter interface{}
	if err := c.sanitizeObject(filter, &sanitizedFilter, "find"); err != nil {
		return err
	}

	// perform the find operation
	cur, err := c.collection.Find(ctx, sanitizedFilter, opts...)
	if err != nil {
		return err
	}

	defer cur.Close(ctx)
	if err := cur.Err(); err != nil {
		return err
	}

	// decode all the results
	var temp []map[string]interface{}
	if err := cur.All(ctx, &temp); err != nil {
		return err
	}

	// finally apply virtuals to the result
	return c.applyVirtuals(temp, target)
}

// FindOne record matching query
func (c *Collection) FindOne(filter interface{}, target interface{}, opts ...*options.FindOneOptions) error {
	tgtVal := reflect.ValueOf(target)
	if tgtVal.Kind() != reflect.Ptr || tgtVal.IsNil() {
		return ErrInvalidTarget
	}

	// create a context
	ctx, cancelFunc := context.WithTimeout(context.Background(), c.queryTimeout*time.Second)
	defer cancelFunc()

	// sanitize the filter
	var sanitizedFilter interface{}
	if err := c.sanitizeObject(filter, &sanitizedFilter, "find"); err != nil {
		return err
	}

	// perform the findOne operation
	result := c.collection.FindOne(ctx, sanitizedFilter, opts...)

	// check for errors in the result
	if err := result.Err(); err != nil {
		return err
	}

	// decode the result to a temp object
	var temp map[string]interface{}
	if err := result.Decode(&temp); err != nil {
		return err
	}

	// finally apply virtuals to the result
	return c.applyVirtuals(temp, target)
}

// InsertOne inserts a document
func (c *Collection) InsertOne(doc interface{}, target interface{}, opts ...*options.InsertOneOptions) error {
	tgtVal := reflect.ValueOf(target)
	if tgtVal.Kind() != reflect.Ptr && !tgtVal.IsNil() {
		return ErrInvalidTarget
	}

	// sanitize the document
	var sanitizedDoc interface{}
	if err := c.sanitizeObject(doc, &sanitizedDoc, "create"); err != nil {
		return err
	}

	// create a context
	ctx, cancelFunc := context.WithTimeout(context.Background(), c.queryTimeout*time.Second)
	defer cancelFunc()

	// insert the document
	result, err := c.collection.InsertOne(ctx, sanitizedDoc, opts...)
	if err != nil {
		return err
	}

	if result.InsertedID == nil {
		return fmt.Errorf("insert failed")
	}

	// get the document and return it
	if !tgtVal.IsNil() {
		return c.FindByID(result.InsertedID, target)
	}
	return nil
}

// FindOneAndUpdate updates a document
func (c *Collection) FindOneAndUpdate(
	filter interface{},
	update interface{},
	target interface{},
	opts ...*options.FindOneAndUpdateOptions,
) error {
	tgtVal := reflect.ValueOf(target)
	// allow target to be nil or a reference
	if tgtVal.Kind() != reflect.Ptr && !tgtVal.IsNil() {
		return ErrInvalidTarget
	} else if update == nil {
		return ErrIncompleteData
	}

	// create context
	ctx, cancelFunc := context.WithTimeout(context.Background(), c.queryTimeout*time.Second)
	defer cancelFunc()

	// sanitize the filter
	var sanitizedFilter interface{}
	if err := c.sanitizeObject(filter, &sanitizedFilter, "update"); err != nil {
		return err
	}

	// sanitize the update
	var sanitizedUpdate interface{}
	if err := c.sanitizeObject(update, &sanitizedUpdate, "update"); err != nil {
		return err
	}

	// perform the update
	result := c.collection.FindOneAndUpdate(ctx, sanitizedFilter, sanitizedUpdate, opts...)
	if err := result.Err(); err != nil {
		return err
	}

	// if the target is not nil get the update
	if !tgtVal.IsNil() {
		var temp map[string]interface{}
		if err := result.Decode(&temp); err != nil {
			return err
		}
		return c.applyVirtuals(temp, target)
	}
	return nil
}

// FindOneAndDelete deletes a document
func (c *Collection) FindOneAndDelete(filter interface{}, target interface{}, opts ...*options.FindOneAndDeleteOptions) error {
	tgtVal := reflect.ValueOf(target)
	// allow target to be nil or a reference
	if tgtVal.Kind() != reflect.Ptr && !tgtVal.IsNil() {
		return ErrInvalidTarget
	}

	// create context
	ctx, cancelFunc := context.WithTimeout(context.Background(), c.queryTimeout*time.Second)
	defer cancelFunc()

	// sanitize the filter
	var sanitizedFilter interface{}
	if err := c.sanitizeObject(filter, &sanitizedFilter, "delete"); err != nil {
		return err
	}

	// perform the update
	result := c.collection.FindOneAndDelete(ctx, sanitizedFilter, opts...)
	if err := result.Err(); err != nil {
		return err
	}

	// if the target is not nil get the update
	if !tgtVal.IsNil() {
		var temp interface{}
		if err := result.Decode(&temp); err != nil {
			return err
		}
		return c.applyVirtuals(temp, target)
	}
	return nil
}

*/
