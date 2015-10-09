// Copyright 2015 Luke Shumaker

package httpentity

import (
	"fmt"
	"net/http"
	"net/url"
)

// For when you're returning a document, with nothing special.
func (req Request) StatusOK(entity NetEntity) Response {
	return Response{
		status:  200,
		Headers: http.Header{},
		entity:  entity,
	}
}

// For when you've created a document with a new URL.
func (req Request) StatusCreated(parent Entity, child_name string) Response {
	child := parent.Subentity(child_name, req)
	if child == nil {
		panic("called StatusCreated, but the subentity doesn't exist")
	}
	handler, ok := child.Methods()["GET"]
	if !ok {
		panic("called StatusCreated, but can't GET the subentity")
	}
	response := handler(req)
	response.Headers.Set("Location", url.QueryEscape(child_name))
	if response.entity == nil {
		panic("called StatusCreated, but GET on subentity doesn't return an entity")
	}
	mimetypes := encoders2mimetypes(response.entity.Encoders())
	u, _ := url.Parse("")
	return Response{
		status:  201,
		Headers: response.Headers,
		// XXX: .entity gets modified by (*Router).route()
		// filled in the rest of the way by Route()
		entity: mimetypes2net(u, mimetypes),
	}
}

// For when you've successfully done something, but have no body to
// return.
func (req Request) StatusNoContent() Response {
	return Response{
		status:  204,
		Headers: http.Header{},
		entity:  nil,
	}
}

// For when you have document in multiple formats, but you're not sure
// which the user wants.
func (req Request) statusMultipleChoices(u *url.URL, mimetypes []string) Response {
	return Response{
		status:  300,
		Headers: http.Header{},
		entity:  mimetypes2net(u, mimetypes),
	}
}

// For when the document the user requested has permantly moved to a
// new address.
func (req Request) StatusMoved(u *url.URL) Response {
	return Response{
		status: 301,
		Headers: http.Header{
			"Location": {u.String()},
		},
		entity: NetString("301: Moved"),
	}
}

// For when the document the user requested is currently found at
// another address, but that may not be the case in the future.
func (req Request) StatusFound(u *url.URL) Response {
	return Response{
		status: 302,
		Headers: http.Header{
			"Location": {u.String()},
		},
		entity: NetString("302: Found"),
	}
}

// For when the *user* has screwed up a request.
func (req Request) StatusBadRequest(err interface{}) Response {
	return Response{
		status:  400,
		Headers: http.Header{},
		entity:  NetString(fmt.Sprintf("400 Bad Request: %v", err)),
	}
}

// For when the user doesn't have permission to see something, either
// because they aren't logged in, or because their account doesn't
// have permission.
func (req Request) StatusUnauthorized(e NetEntity) Response {
	return Response{
		status: 401,
		Headers: http.Header{
			"WWW-Authenticate": {"TODO: long-term"},
		},
		entity: e,
	}
}

func (req Request) statusNotFound() Response {
	return Response{
		status:  404,
		Headers: http.Header{},
		entity:  NetString("404 Not Found"),
	}
}

func (req Request) statusMethodNotAllowed(methods string) Response {
	return Response{
		status:  405,
		Headers: http.Header{},
		entity:  NetString("405 Method Not Allowed"),
	}
}

func (req Request) statusNotAcceptable(u *url.URL, mimetypes []string) Response {
	return Response{
		status:  406,
		Headers: http.Header{},
		entity:  mimetypes2net(u, mimetypes),
	}
}

// For when the user asked us to make a change conflicting with the
// current state of things.
func (req Request) StatusConflict(entity NetEntity) Response {
	return Response{
		status:  409,
		Headers: http.Header{},
		entity:  entity,
	}
}

func (req Request) statusUnsupportedMediaType() Response {
	return Response{
		status:  415,
		Headers: http.Header{},
		entity:  NetString("415 Unsupported Media Type"),
	}
}

func (req Request) statusInternalServerError(err interface{}) Response {
	return Response{
		status: 500,
		Headers: http.Header{
			"Content-Type": {"text/plain; charset=utf-8"},
		},
		entity: NetString(fmt.Sprintf("500 Internal Server Error: %v", err)),
	}
}
