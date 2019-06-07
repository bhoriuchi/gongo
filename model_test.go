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
	Name        string `json:"name"       required:"true"  unique:"true"`
	Description string `json:"description"`
}

func TestModel(t *testing.T) {
	// create a client
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	g := New("gongo-test", clientOptions)

	// define schema
	testFooSchema := NewSchema(&testFoo{})
	testFooSchema.Virtual(&VirtualConfig{
		Name: "id",
		Get:  helpers.VirtualGetObjectIDAsHexString("_id"),
		Set:  helpers.VirtualSetObjectID("_id"),
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
}
