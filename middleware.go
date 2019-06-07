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
type PreMiddlewareFunc func(documentOrQuery bson.M, model *Model) error

// PostMiddlewareFunc middleware for post
type PostMiddlewareFunc func(document bson.M, model *Model, err error) error

// middleware is kept as a map with integer keys to ensure
// middlewares are called in the order they were registered
type middlewareConfig struct {
	pre  map[int]*PreMiddleware
	post map[int]*PostMiddleware
}

// Pre adds pre middleware
func (c *Schema) Pre(operation string, handler PreMiddlewareFunc, async ...*bool) *Schema {
	isAsync := false
	if len(async) > 0 {
		a := async[0]
		isAsync = *a
	}
	switch operation {
	case "save", "validate", "remove", "init", "count", "deleteMany",
		"deleteOne", "find", "findOne", "findOneAndDelete", "findOneAndRemove",
		"findOneAndUpdate", "update", "updateOne", "updateMany":

		c.middleware.pre[len(c.middleware.pre)] = &PreMiddleware{
			Operation: operation,
			Handler:   handler,
			Async:     isAsync,
		}
		return c
	}
	return c
}

// Post adds post middleware
func (c *Schema) Post(operation string, handler PostMiddlewareFunc, async ...*bool) *Schema {
	isAsync := false
	if len(async) > 0 {
		a := async[0]
		isAsync = *a
	}

	switch operation {
	case "save", "validate", "remove", "init", "count", "deleteMany",
		"deleteOne", "find", "findOne", "findOneAndDelete", "findOneAndRemove",
		"findOneAndUpdate", "update", "updateOne", "updateMany":
		c.middleware.post[len(c.middleware.post)] = &PostMiddleware{
			Operation: operation,
			Handler:   handler,
			Async:     isAsync,
		}
		return c
	}
	return c
}

// apply the pre middleware
func (c *Schema) applyPreMiddleware(operation string, documentOrQuery bson.M, model *Model) error {
	// loop using integer iterator to keep order
	for i := 0; i < len(c.middleware.pre); i++ {
		if mw, ok := c.middleware.pre[i]; ok {
			if mw.Operation == operation {
				if !mw.Async {
					if err := mw.Handler(documentOrQuery, model); err != nil {
						return err
					}
				} else {
					// run async as goroutine
					go mw.Handler(documentOrQuery, model)
				}
			}
		}
	}
	return nil
}

// apply the post middleware
func (c *Schema) applyPostMiddleware(operation string, document bson.M, model *Model, err error) error {
	// loop using integer iterator to keep order
	for i := 0; i < len(c.middleware.post); i++ {
		if mw, ok := c.middleware.post[i]; ok {
			if mw.Operation == operation {
				if !mw.Async {
					if err := mw.Handler(document, model, err); err != nil {
						return err
					}
				} else {
					// run async as goroutine
					go mw.Handler(document, model, err)
				}
			}
		}
	}
	return nil
}
