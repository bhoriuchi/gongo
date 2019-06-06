package helpers

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VirtualSetObjectIDFromHexString sets a virtual object id from a hex string
func VirtualSetObjectIDFromHexString(value interface{}, doc bson.M) error {
	if value == nil {
		return nil
	}
	id := value.(string)
	if id == "" {
		return fmt.Errorf("no hex ID specified")
	}
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	doc["_id"] = objectID
	return nil
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
		oid := v.(primitive.ObjectID)
		return oid.Hex(), nil
	}
}

// VirtualSetNoop does nothing, you're welcome
func VirtualSetNoop(value interface{}, doc bson.M) error {
	return nil
}
