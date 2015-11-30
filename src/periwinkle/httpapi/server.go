// Copyright 2015 Luke Shumaker
// Copyright 2015 Zhandos Suleimenov

package httpapi

import (
	he "httpentity"
	"httpentity/heutil"
	"io"
	"log"
	"net"
	"net/http"
	"periwinkle"
	"periwinkle/domain_handlers"
	"stoppable"
)

func MakeServer(socket net.Listener, cfg *periwinkle.Cfg) *stoppable.HTTPServer {
	stdDecoders := map[string]func(io.Reader, map[string]string) (interface{}, error){
		"application/x-www-form-urlencoded": heutil.DecoderFormURLEncoded,
		"multipart/form-data":               heutil.DecoderFormData,
		"application/json":                  heutil.DecoderJSON,
		"application/json-patch+json":       heutil.DecoderJSONPatch,
		"application/merge-patch+json":      heutil.DecoderJSONMergePatch,
	}
	stdMiddlewares := []he.Middleware{
		MiddlewarePostHack,
		MiddlewareDatabase(cfg),
		MiddlewareSession,
	}
	mux := http.NewServeMux()
	// The main REST API
	mux.Handle("/v1/", he.Router{
		Prefix:         "/v1/",
		Root:           NewDirRoot(),
		Decoders:       stdDecoders,
		Middlewares:    stdMiddlewares,
		Stacktrace:     cfg.Debug,
		LogRequest:     cfg.Debug,
		TrustForwarded: cfg.TrustForwarded,
	}.Init())
	// URL shortener service
	mux.Handle("/s/", he.Router{
		Prefix:         "/s/",
		Root:           NewDirShortURLs(),
		Decoders:       stdDecoders,
		Middlewares:    stdMiddlewares,
		Stacktrace:     cfg.Debug,
		LogRequest:     cfg.Debug,
		TrustForwarded: cfg.TrustForwarded,
	}.Init())

	// The static web UI
	mux.Handle("/webui/", http.StripPrefix("/webui/", http.FileServer(cfg.WebUIDir)))

	smsCallbackServer := domain_handlers.SmsCallbackServer{}
	go func() {
		err := smsCallbackServer.Serve()
		if err != nil {
			log.Printf("Could not serve SmsCallbackServer: %v\n", err)
		}
	}()

	// External API callbacks
	mux.Handle("/callbacks/twilio-sms", http.HandlerFunc(smsCallbackServer.ServeHTTP))

	// Make the server
	return &stoppable.HTTPServer{
		Server: http.Server{Handler: mux},
		Socket: socket,
	}
}