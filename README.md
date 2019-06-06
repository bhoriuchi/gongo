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
  "github.com/bhoriuchi/gongo"
  "go.mongodb.org/mongo-driver/mongo/options"
)

type TestFoo struct {
	ID          string `json:"id"           mongo_type:"ObjectId" mongo_virtual:"$_id"`
	Name        string `json:"name"         required:"true"       unique:"true"`
	Description string `json:"description"`
}

func main() {
  clientOptions := options.Client().ApplyURI("mongodb://mongo.mydomain.com:27017")
  g := gongo.New("mydbname", clientOptions)
}
```