package helpers

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VirtualSetObjectID sets an object id
func VirtualSetObjectID(fieldName string) func(value interface{}, doc bson.M) error {
	return func(value interface{}, doc bson.M) error {
		if fieldName == "" {
			return fmt.Errorf("VirtualSetObjectID has no field name specified")
		}
		if IsObjectID(value) {
			doc[fieldName] = value
			return nil
		}

		id := fmt.Sprintf("%v", value)
		if id == "" {
			return fmt.Errorf("no object id specified")
		}
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return err
		}
		doc[fieldName] = objectID

		return nil
	}
}

// VirtualGetObjectIDAsHexString returns the specified field
// containing an ObjectID as a hex string
func VirtualGetObjectIDAsHexString(fieldName string) func(doc bson.M) (interface{}, error) {
	return func(doc bson.M) (interface{}, error) {
		if fieldName == "" {
			return nil, fmt.Errorf("no field name specified")
		}
		v, ok := doc[fieldName]
		if !ok {
			return nil, fmt.Errorf("field %q not found", fieldName)
		} else if v == nil {
			return nil, fmt.Errorf("field %q not set", fieldName)
		}

		// if its an object id convert to hex
		if IsObjectID(v) {
			oid := v.(primitive.ObjectID)
			return oid.Hex(), nil
		}

		// otherwise just try to return the string
		return fmt.Sprintf("%v", v), nil
	}
}

// VirtualSetNoop does nothing, you're welcome
func VirtualSetNoop(value interface{}, doc bson.M) error {
	return nil
}
