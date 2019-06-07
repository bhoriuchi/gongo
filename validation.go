package gongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

// ValidateFunc a validator function
type ValidateFunc func(value interface{}, fieldName string, doc bson.M) error

// WithValidator registers a validator
func (c *Gongo) WithValidator(name string, handler ValidateFunc) *Gongo {
	if name == "" {
		c.log.Error("WithValidator: no validator name provided")
		return c
	} else if _, ok := c.validators[name]; ok {
		c.log.Errorf("WithValidator: validator with name %q already registered", name)
		return c
	}
	c.validators[name] = handler
	return c
}

// Validate validates a model
func (c *Model) Validate() error {
	// perform the required validation
	if err := c.checkRequired(); err != nil {
		return err
	}

	// now perform custom validations
	for _, v := range *c.fieldTagMap {
		validations, foundValidate := v.getList(c.gongo.fieldTagDef.Get("validate"))
		name, foundName := v.getString(c.gongo.fieldTagDef.Get("name"))

		if !foundValidate || !foundName {
			continue
		}

		doc := *c.document
		value, ok := doc[name]
		if !ok {
			value = nil
		}

		// perform each validation defined
		for _, vName := range validations {
			// find the validator
			if validator, ok := c.gongo.validators[vName]; ok {
				if err := validator(value, name, doc); err != nil {
					return err
				}
			} else {
				c.gongo.log.Warnf("failed to find validator %q", vName)
			}
		}
	}

	return nil
}

func (c *Model) checkRequired() error {
	for _, v := range *c.fieldTagMap {
		required, foundRequired := v.getString(c.gongo.fieldTagDef.Get("required"))
		name, foundName := v.getString(c.gongo.fieldTagDef.Get("name"))

		// verify that the field tags are set up correctly
		if !foundRequired || !foundName || required == "false" || required == "0" {
			continue
		}

		// get the value
		doc := *c.document
		value, ok := doc[name]
		if !ok {
			return fmt.Errorf("required field %q not found", name)
		}

		// if required is truthy continue
		if required == "true" || required == "1" {
			continue
		}

		// otherwise the required field should point to a validator
		// function
		if validator, ok := c.gongo.validators[required]; ok {
			if err := validator(value, name, doc); err != nil {
				return err
			}
		} else {
			c.gongo.log.Warnf("failed to find validator %q", required)
		}
	}
	return nil
}
