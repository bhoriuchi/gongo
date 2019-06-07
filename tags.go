package gongo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bhoriuchi/gongo/helpers"
)

// required tags
var requiredTagIDs = []string{"name", "required", "unique", "validate"}

// FieldTagMap a map of fields containg tag mappings
type fieldTagMap map[string]*tagMap

// TagMap a map of tags to tag names
type tagMap map[string]string

// Get gets a tag value from the map
func (c *tagMap) getString(tagName string) (string, bool) {
	tm := *c
	val, ok := tm[tagName]
	return val, ok
}

// gets a list of values
func (c *tagMap) getList(tagName string) ([]string, bool) {
	val, ok := c.getString(tagName)
	if !ok || val == "" {
		return nil, ok
	}
	list := strings.Split(val, ",")
	return list, ok
}

// gets the bool value from the tag map
func (c *tagMap) getBool(tagName string) (bool, bool) {
	val, ok := c.getString(tagName)
	if !ok {
		return false, ok
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return false, ok
	}
	return b, ok
}

// FindOneByName finds a tag map by match the tag with value
func (c *fieldTagMap) findOneByName(tagName, tagValue string) *tagMap {
	fm := *c
	for _, tm := range fm {
		if _, ok := tm.getString(tagName); ok {
			return tm
		}
	}
	return nil
}

// MapStructTags maps struct tags to their values
func mapStructTags(structure interface{}, tagNames *[]string) *fieldTagMap {
	fm := make(fieldTagMap)

	if structure == nil {
		return &fm
	}
	sv := helpers.GetElement(structure)
	st := sv.Type()

	for i := 0; i < sv.NumField(); i++ {
		field := st.Field(i)
		tm := make(tagMap)
		for _, name := range *tagNames {
			if val, ok := field.Tag.Lookup(name); ok {
				tm[name] = val
			}
		}

		// set the tag map as a field on the field map
		fm[field.Name] = &tm
	}

	return &fm
}

// FieldTagDefinition defines the field tags
// each value is used to lookup the tag name
// responsible for the named configuration
type FieldTagDefinition map[string]string

// Validate validates the field tag definition
func (c *FieldTagDefinition) Validate() error {
	m := *c
	valueMap := make(map[string]bool)

	// validate all tag ids have values and there are
	// no duplicate values
	for id, tag := range m {
		if tag == "" {
			return fmt.Errorf("tag for %q cannot have an empty value", id)
		} else if _, ok := valueMap[tag]; ok {
			return fmt.Errorf("tag %q has been defined multiple times", tag)
		}
		valueMap[tag] = true
	}

	// validate required tags exist
	for _, id := range requiredTagIDs {
		if _, ok := m[id]; !ok {
			return fmt.Errorf("required tag id %q not defined", id)
		}
	}

	return nil
}

// Tags retuns all the field tag values
func (c *FieldTagDefinition) Tags() *[]string {
	tags := make([]string, 0)
	for _, tag := range *c {
		tags = append(tags, tag)
	}
	return &tags
}

// Get gets a tag
func (c *FieldTagDefinition) Get(id string) string {
	m := *c
	return m[id]
}

// DefaultFieldTagDefinition returns a default field taf definition
func DefaultFieldTagDefinition() *FieldTagDefinition {
	return &FieldTagDefinition{
		"name":     "json",
		"required": "required",
		"unique":   "unique",
		"validate": "validate",
	}
}
