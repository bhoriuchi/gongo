# gongo
Kind of like [mongoose.js](https://mongoosejs.com) for Golang... but not really

## Overview

* Loosly based off of [mongoose.js](https://mongoosejs.com)
* Uses the official mongodb driver for go
* Define models using structs with tagged fields
* If you are snoby about using [reflect](https://golang.org/pkg/reflect/), this is not the package for you
* Has a stupid name because all the good ones were taken
* PRs welcome

## Installation

```sh
go get github.com/bhoriuchi/gongo
```

## Example

```go
package main

import (
  "os"

  "github.com/bhoriuchi/gongo"
  "github.com/bhoriuchi/gongo/helpers"
  "go.mongodb.org/mongo-driver/bson"
  "go.mongodb.org/mongo-driver/mongo/options"
)

type TestFoo struct {
	ID          string `json:"id"`
	Name        string `json:"name" required:"true" unique:"true"`
	Description string `json:"description"`
}

func main() {
  // create a new instance
  clientOptions := options.Client().ApplyURI("mongodb://mongo.mydomain.com:27017")
  g := gongo.New("mydbname", clientOptions)

	// define schema and add virtuals
  testFooSchema := NewSchema(&testFoo{})
  testFooSchema.Virtual(&VirtualConfig{
    Name: "id",
    Get:  helpers.VirtualGetObjectIDAsHexString("_id"),
    Set:  helpers.VirtualSetObjectIDFromHexString,
  })

  // create model
  foo, _ := g.Model(testFooSchema)

  // connect to the database
  if err := g.Connect(); err != nil {
    t.Errorf("failed to connect: %s", err.Error())
    return
  }

  // perform insert
  var insertResult testFoo
  if err := foo.InsertOne(bson.M{"name": "bar"}, &insertResult); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  fmt.Println(testFoo)
  
  // perform find
	var findResult testFoo
	filter := &bson.M{
		"name": "test1",
	}
	if err := foo.FindOne(filter, &findResult); err != nil {
    fmt.Println(err)
		os.Exit(1)
	}
  fmt.Println(findResult)
}
```