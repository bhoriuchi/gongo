package gongo

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestWalkRemoveUndefined(t *testing.T) {
	g := New()
	fooSchema := Schema{
		gongo: g,
		Fields: SchemaFieldMap{
			"name": {
				Type:     StringType,
				Required: true,
			},
			"description": {
				Type:    StringType,
				Default: "bar",
			},
		},
	}

	fooDoc := bson.M{
		"name": "foo",
		"bar":  "baz",
	}

	if err := fooSchema.init(); err != nil {
		t.Error(err)
		return
	}

	actual, err := fooSchema.walk(fooDoc, []string{}, &walkOptions{
		applySetters:     true,
		applyDefaults:    true,
		castObjectID:     true,
		validateTypes:    true,
		validateCustom:   true,
		validateRequired: true,
	})

	if err != nil {
		t.Error(err)
		return
	}

	expected := &bson.M{"name": "foo", "description": "bar"}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %v, actual %v", expected, actual)
		return
	}
}

func TestWalkNested(t *testing.T) {
	g := New()
	barSchema := Schema{
		gongo: g,
		Fields: SchemaFieldMap{
			"name": {
				Type:     StringType,
				Required: true,
			},
			"description": {
				Type:    StringType,
				Default: "bar",
			},
		},
	}
	fooSchema := Schema{
		gongo: g,
		Fields: SchemaFieldMap{
			"name": {
				Type:     StringType,
				Required: true,
			},
			"bar": {
				Type: barSchema,
			},
		},
	}

	fooDoc := bson.M{
		"name": "foo",
		"bar": bson.M{
			"name":        "baz",
			"description": "qux",
			"ignore":      "this",
		},
	}

	if err := fooSchema.init(); err != nil {
		t.Error(err)
		return
	}

	actual, err := fooSchema.walk(fooDoc, []string{}, &walkOptions{
		applySetters:     true,
		applyDefaults:    true,
		castObjectID:     true,
		validateTypes:    true,
		validateCustom:   true,
		validateRequired: true,
	})

	if err != nil {
		t.Error(err)
		return
	}

	expected := &bson.M{
		"name": "foo",
		"bar": &bson.M{
			"name":        "baz",
			"description": "qux",
		},
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %v, actual %v", expected, actual)
		return
	}
}
