package gongo

import "fmt"

// FieldTagMap a map of fields containg tag mappings
type fieldTagMap map[string]*tagMap

// TagMap a map of tags to tag names
type tagMap map[string]string

// Get gets a tag value from the map
func (c *tagMap) get(tagName string) *string {
	tm := *c
	val, ok := tm[tagName]
	if !ok {
		return nil
	}
	return &val
}

// Field gets the tagmap at a specific field
func (c *fieldTagMap) field(fieldName string) *tagMap {
	fm := *c
	return fm.findOneByName("json", fieldName)
}

// Get gets a tag from a specific field
func (c *fieldTagMap) Get(fieldName, tagName string) *string {
	if tm := c.field(fieldName); tm != nil {
		return tm.get(tagName)
	}
	return nil
}

// FindOneByName finds a tag map by match the tag with value
func (c *fieldTagMap) findOneByName(tagName, tagValue string) *tagMap {
	fm := *c
	for _, tm := range fm {
		val := tm.get(tagName)
		if val != nil && *val == tagValue {
			return tm
		}
	}
	return nil
}

// Values returns the values of a tag name from all fields
func (c *fieldTagMap) values(tagName string) *[]string {
	v := make([]string, 0)
	fm := *c
	for _, tm := range fm {
		value := tm.get(tagName)
		if value != nil {
			fieldName := tm.get("json")
			v = append(v, *fieldName)
		}
	}

	return &v
}

// MapStructTags maps struct tags to their values
func mapStructTags(structure interface{}, tagNames *[]string) *fieldTagMap {
	fm := make(fieldTagMap)

	if structure == nil {
		return &fm
	}
	sv := getElement(structure)
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

// gets the actual field name if field is a virtual
func getActualFieldName(tagMap *tagMap) *string {
	virtual := tagMap.get("mongo_virtual")
	if virtual != nil && *virtual != "" {
		vField := *virtual
		if vField[:1] == "$" {
			name := vField[1:]
			return &name
		}
	} else {
		fieldName := tagMap.get("json")
		if fieldName != nil && *fieldName != "" {
			return fieldName
		}
	}
	return nil
}

// FieldTagDefinition defines the field tags
type FieldTagDefinition struct {
	Name      string
	OmitIf    string
	Required  string
	Unique    string
	MongoType string
	PrimaryID string
}

// Validate validates the field tag definition
func (c *FieldTagDefinition) Validate() error {
	tagMap := make(map[string]bool)
	v := getElement(c)
	for i := 0; i < v.NumField(); i++ {
		value := v.Field(i).Interface().(string)
		if value == "" {
			return fmt.Errorf("invalid tag value found")
		} else if _, ok := tagMap[value]; ok {
			return fmt.Errorf("tag value %s used multiple times", value)
		}
		tagMap[value] = true
	}
	return nil
}

// Tags retuns all the field tag values
func (c *FieldTagDefinition) Tags() *[]string {
	tags := make([]string, 0)
	v := getElement(c)
	for i := 0; i < v.NumField(); i++ {
		value := v.Field(i).Interface().(string)
		tags = append(tags, value)
	}
	return &tags
}

// DefaultFieldTagDefinition returns a default field taf definition
func DefaultFieldTagDefinition() *FieldTagDefinition {
	return &FieldTagDefinition{
		Name:      "json",
		OmitIf:    "omit_if",
		Required:  "required",
		Unique:    "unique",
		MongoType: "mongo_type",
		PrimaryID: "primary_id",
	}
}
