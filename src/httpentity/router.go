// Copyright 2015 Luke Shumaker

package httpentity

import (
	"bitbucket.org/ww/goautoneg"
	"fmt"
	"log"
	"mime"
	"net/url"
	"path"
	"runtime"
	"strings"
)

func normalizeURL(u1 *url.URL) (u *url.URL, mimetype string) {
	u, _ = u1.Parse("") // normalize
	// the file extension overrides the Accept: header
	if ext := path.Ext(u.Path); ext != "" {
		mimetype = mime.TypeByExtension(ext)
		u.Path = strings.TrimSuffix(u.Path, ext)
	}
	// add a trailing slash if there isn't one (so that relative
	// child URLs don't go to the parent)
	if !strings.HasSuffix(u.Path, "/") {
		u.Path = u.Path + "/"
	}
	return
}

// assumes that the url has already been passed to normalizeURL()
func (router *Router) route(request Request, u *url.URL) Response {
	if router.LogRequest {
		log.Printf("route %s %q %#v\n", request.Method, u.String(), request)
	}

	handler := router.defaultHandler
	for i := 0; i < len(router.Middlewares); i++ {
		handler = middlewareHolder{
			middleware:  router.Middlewares[len(router.Middlewares)-1-i],
			nextHandler: handler,
		}.handler
	}
	return handler(request, u)
}

// assumes that the url has already been passed to normalizeURL()
func (r *Router) finish(req Request, u *url.URL, res *Response) {
	// generate a 500 error if we paniced
	if err := recover(); err != nil {
		reason := err
		if r.Stacktrace {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			reason = fmt.Sprintf("%T(%#v) => %v\n\n%s\n", err, err, err, string(buf))
			for _, line := range strings.Split(reason.(string), "\n") {
				log.Println(line)
			}
		}
		*res = statusInternalServerError(reason)
	}
	// figure out the content type of the response
	if res.Entity != nil && res.Headers.Get("Content-Type") == "" {
		encoders := res.Entity.Encoders()
		mimetypes := encoders2mimetypes(encoders)
		accept := req.Headers.Get("Accept")
		if len(encoders) > 1 && accept == "" {
			*res = statusMultipleChoices(u, mimetypes)
		} else {
			// TODO: long term: In the event of a tie,
			// goautoneg returns the first match in the
			// mimetypes array, which in our case is
			// essentially random.  Instead, we should
			// return an HTTP 300 Multiple Choices.  This
			// means forking or re-implementing goautoneg.
			mimetype := goautoneg.Negotiate(accept, mimetypes)
			if mimetype == "" {
				*res = statusNotAcceptable(u, mimetypes)
			} else {
				res.Headers.Set("Content-Type", mimetype+"; charset=utf-8")
			}
		}
	}
}
