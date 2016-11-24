package command

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"
	"golang.org/x/net/context"
)

type HTTPHandlers struct {
	NewCommandHandler http.Handler
}

func MakeHTTPHandlers(ctx context.Context, endpoints Endpoints, opts ...httptransport.ServerOption) HTTPHandlers {
	h := HTTPHandlers{
		NewCommandHandler: httptransport.NewServer(
			ctx,
			endpoints.NewCommandEndpoint,
			decodeRequest,
			encodeResponse,
			opts...,
		),
	}
	return h
}

type errorer interface {
	error() error
}

type statuser interface {
	status() int
}

// EncodeError is used by the HTTP transport to encode service errors in HTTP.
// The EncodeError should be passed to the Go-Kit httptransport as the
// ServerErrorEncoder to encode error responses with JSON.
func EncodeError(ctx context.Context, err error, w http.ResponseWriter) {
	// unwrap Go-Kit Error
	var domain string
	if e, ok := err.(httptransport.Error); ok {
		err = e.Err
		domain = e.Domain
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	switch domain {
	case httptransport.DomainDecode:
		w.WriteHeader(http.StatusBadRequest)
	case httptransport.DomainDo:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(codeFromErr(err))
	}
	enc.Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func codeFromErr(err error) int {
	switch err {
	case errEmptyRequest:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

type errorWrapper struct {
	Error string `json:"error"`
}

func decodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req newCommandRequest
	err := json.NewDecoder(io.LimitReader(r.Body, 10000)).Decode(&req)
	return req, err
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {

	if e, ok := response.(errorer); ok && e.error() != nil {
		EncodeError(ctx, e.error(), w)
		return nil
	}

	if s, ok := response.(statuser); ok {
		w.WriteHeader(s.status())
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(response)
}
