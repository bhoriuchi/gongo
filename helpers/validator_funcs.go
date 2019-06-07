package helpers

import (
	"fmt"
	"regexp"

	"go.mongodb.org/mongo-driver/bson"
)

var alphaNumeric = regexp.MustCompile("^[0-9A-Za-z]+$")

// ValidatorAlphaNumeric performs an alpha-numeric validation on a value
func ValidatorAlphaNumeric(value interface{}, fieldName string, doc bson.M) error {
	if value == nil || !IsString(value) {
		return fmt.Errorf("field %q failed alpha-numeric validation", fieldName)
	}
	if !alphaNumeric.MatchString(value.(string)) {
		return fmt.Errorf("field %q failed alpha-numeric validation", fieldName)
	}
	return nil
}
