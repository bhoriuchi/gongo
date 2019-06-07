package gongo

import "go.mongodb.org/mongo-driver/bson"

// PreMiddleware a middleware
type PreMiddleware struct {
	Operation string
	Handler   PreMiddlewareFunc
	Async     bool
}

// PostMiddleware a middleware
type PostMiddleware struct {
	Operation string
	Handler   PostMiddlewareFunc
	Async     bool
}

// PreMiddlewareFunc middleware for document
type PreMiddlewareFunc func(documentOrQuery bson.M) error

// PostMiddlewareFunc middleware for post
type PostMiddlewareFunc func(document bson.M, err error) error

type middlewareConfig struct {
	pre  []*PreMiddleware
	post []*PostMiddleware
}

// Pre adds pre middleware
func (c *Schema) Pre(operation string, handler PreMiddlewareFunc, async ...*bool) {
	isAsync := false
	if len(async) > 0 {
		a := async[0]
		isAsync = *a
	}
	switch operation {
	case "save", "validate", "remove", "init", "count", "deleteMany",
		"deleteOne", "find", "findOne", "findOneAndDelete", "findOneAndRemove",
		"findOneAndUpdate", "update", "updateOne", "updateMany":
		c.middleware.pre = append(c.middleware.pre, &PreMiddleware{
			Operation: operation,
			Handler:   handler,
			Async:     isAsync,
		})
		return
	}
}

// Post adds post middleware
func (c *Schema) Post(operation string, handler PostMiddlewareFunc, async ...*bool) {
	isAsync := false
	if len(async) > 0 {
		a := async[0]
		isAsync = *a
	}

	switch operation {
	case "save", "validate", "remove", "init", "count", "deleteMany",
		"deleteOne", "find", "findOne", "findOneAndDelete", "findOneAndRemove",
		"findOneAndUpdate", "update", "updateOne", "updateMany":
		c.middleware.post = append(c.middleware.post, &PostMiddleware{
			Operation: operation,
			Handler:   handler,
			Async:     isAsync,
		})
		return
	}

}
