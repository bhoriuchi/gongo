package gongo

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bhoriuchi/gongo/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type testFoo struct {
	ID          string `json:"id"         primary_id:"true"`
	Name        string `json:"name"       required:"true"  unique:"true" validate:"alphanumeric"`
	Description string `json:"description"`
}

func TestModel(t *testing.T) {
	// create a client
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	g := New("gongo-test", clientOptions).
		WithLogger(&SimpleLogger{}).
		WithValidator("alphanumeric", helpers.ValidatorAlphaNumeric)

	// define schema
	testFooSchema := NewSchema(&testFoo{})
	testFooSchema.
		Virtual(&VirtualConfig{
			Name: "id",
			Get:  helpers.VirtualGetObjectIDAsHexString("_id"),
			Set:  helpers.VirtualSetObjectID("_id"),
		}).
		Pre("save", func(doc bson.M, model *Model) error {
			name := model.Get("name")
			model.Set("name", fmt.Sprintf("%s1", name))
			return nil
		}).
		Pre("save", func(doc bson.M, model *Model) error {
			name := model.Get("name")
			model.Set("name", fmt.Sprintf("%s2", name))
			return nil
		})

	// create models
	foo, err := g.Model(testFooSchema)
	if err != nil {
		t.Errorf("failed to create new model: %s", err.Error())
		return
	}

	// connect to the database
	if err := g.Connect(); err != nil {
		t.Errorf("failed to connect: %s", err.Error())
		return
	}

	f := foo.New(nil)
	if err := f.Hydrate(&bson.M{"id": "5cfa9b69cf6f55fc8d3f40c8"}); err != nil {
		t.Errorf("%s", err.Error())
		return
	}

	f.Set("name", "bobo")
	if err := f.Save(); err != nil {
		t.Errorf("%s", err.Error())
		return
	}

	var hydratedFoo testFoo
	if err := f.Decode(&hydratedFoo); err != nil {
		t.Errorf("%s", err.Error())
		return
	}
	j, err := json.MarshalIndent(hydratedFoo, "", "  ")
	fmt.Printf("HYDRATED\n%s\n", j)

	/*
			f := foo.New(&bson.M{
				// "name":        "blah",
				"description": "de dah",
			}).Set("name", "blah")


		if err := f.Save(); err != nil {
			t.Errorf("save error: %s", err.Error())
			return
		}

		fmt.Println(f.document)

			// perform an insert
			var insertResult testFoo
			if err := foo.InsertOne(bson.M{"name": "bar"}, &insertResult); err != nil {
				t.Errorf("insert error: %s", err.Error())
				return
			}
			j1, err := json.MarshalIndent(insertResult, "", "  ")
			if err != nil {
				t.Errorf("marshal error: %s", err.Error())
				return
			}
			fmt.Printf("%s\n", j1)

			// perform some operations
			var findResult testFoo
			filter := &bson.M{
				"name": "test1",
			}
			if err := foo.FindOne(filter, &findResult); err != nil {
				t.Errorf("failed to findOne: %s", err.Error())
				return
			}
			j, err := json.MarshalIndent(findResult, "", "  ")
			if err != nil {
				t.Errorf("marshal error: %s", err.Error())
				return
			}
			fmt.Printf("%s\n", j)
	*/
}
