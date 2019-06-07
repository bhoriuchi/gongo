package gongo

import (
	"go.mongodb.org/mongo-driver/bson"
)

// VirtualFieldMap a map of virtual types
type VirtualFieldMap map[string]*VirtualConfig

// VirtualGetFunc for resolving virtual
type VirtualGetFunc func(doc bson.M) (interface{}, error)

// VirtualSetFunc for resolving virtual
type VirtualSetFunc func(value interface{}, doc bson.M) error

// VirtualConfig defines the virtual config
type VirtualConfig struct {
	Name string
	Get  VirtualGetFunc
	Set  VirtualSetFunc
}

// Virtual adds a virtual field config
func (c *Schema) Virtual(config *VirtualConfig) {
	if config == nil || config.Name == "" {
		return
	}
	if c.virtuals == nil {
		c.virtuals = &VirtualFieldMap{}
	}
	virtuals := *c.virtuals
	if _, ok := virtuals[config.Name]; ok {
		return
	}
	virtuals[config.Name] = config
}

// apply virtuals getters
func (c *Schema) applyVirtualGetters(doc bson.M) error {
	for _, v := range *c.virtuals {
		value, err := v.Get(doc)
		if err != nil {
			return err
		}
		doc[v.Name] = value
	}
	return nil
}
