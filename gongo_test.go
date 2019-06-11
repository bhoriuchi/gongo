package gongo

import (
	"encoding/json"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestGongo(t *testing.T) {
	var dbURI = "mongodb://localhost:27017"
	g := New(&Options{DefaultDatabase: "gongo-test", FieldTag: "json"})

	barSchema := Schema{
		Fields: SchemaFieldMap{
			"name": {
				Type:     StringType,
				Required: true,
				Validate: &[]ValidatorFunc{
					func(value interface{}) error {
						if value.(string) == "" {
							return fmt.Errorf("name cannot be empty")
						}
						return nil
					},
				},
			},
		},
	}

	fooSchema := Schema{
		Fields: SchemaFieldMap{
			"name": {
				Type:     StringType,
				Required: true,
			},
			"bar": {
				Type:     barSchema,
				Required: true,
			},
			"description": {
				Type:    StringType,
				Default: "nobody likes these",
			},
		},
	}

	foo, err := g.Model("Foo", &fooSchema)
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}

	if err := g.Connect(dbURI); err != nil {
		t.Errorf("%s", err.Error())
		return
	}

	doc, err := foo.New(bson.M{
		"name":        "stuff",
		"description": "barz",
		"bar": bson.M{
			"name": "stuff",
			"qux":  "blah",
		},
		"baz": 1,
	})
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}

	if err := doc.Set("bar.name", "buzz"); err != nil {
		t.Errorf("%s", err.Error())
		return
	}

	if err := doc.Save(); err != nil {
		t.Errorf("%s", err.Error())
		return
	}

	docs, err := foo.Find(bson.M{})
	if err != nil {
		t.Errorf("%s", err.Error())
		return
	}

	var res []bson.M
	if err := docs.Decode(&res); err != nil {
		t.Errorf("%s", err.Error())
		return
	}

	j, _ := json.MarshalIndent(res, "", "  ")

	fmt.Printf("%s\n", j)

}
