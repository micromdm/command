package command

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	httptransport "github.com/go-kit/kit/transport/http"

	"github.com/micromdm/command/service/mock"
	"github.com/micromdm/mdm"
)

func TestNewCommandHTTP(t *testing.T) {
	client := setup(t)
	defer client.Close()
	request := mustMarshalJSONRequest(t, &mdm.CommandRequest{
		RequestType: "SomeMDMCommand",
		UDID:        "some-device",
	})

	var httpTests = []struct {
		name         string
		method       mock.NewCommandFunc
		request      io.Reader
		expectStatus int
	}{
		{
			name:         "happy_path",
			method:       mock.ReturnMockPayload,
			request:      request,
			expectStatus: http.StatusCreated,
		},
		{
			name:         "bad_request",
			method:       mock.ReturnMockPayload,
			request:      mustMarshalJSONRequest(t, new(mdm.CommandRequest)),
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "limit_reader",
			method:       mock.ReturnMockPayload,
			request:      neverEnding('a'),
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range httpTests {
		t.Run(tt.name, func(t *testing.T) {
			client.svc.NewCommandFunc = tt.method
			resp := client.Do(t, "POST", tt.request)
			if want, have := tt.expectStatus, resp.StatusCode; want != have {
				t.Fatalf("want %d, have %d", want, have)
			}
			var p struct {
				Payload *mdm.Payload
				Err     string
			}
			if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
				t.Fatalf("failed to decode json request")
			}
			if !client.svc.NewCommandInvoked &&
				resp.StatusCode == http.StatusCreated {
				t.Errorf("request suceeded without invoking service method.")
			}
		})
	}
}

// a never ending io.Reader for testing that the server terminates a request
// with a too large body.
type neverEnding byte

func (b neverEnding) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}

func mustMarshalJSONRequest(t *testing.T, req interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	return buf

}

type client struct {
	*httptest.Server
	svc    *mock.CommandService
	client *http.Client
}

func (s client) Do(t *testing.T, method string, body io.Reader) *http.Response {
	req, err := http.NewRequest(method, s.URL, body)
	if err != nil {
		t.Fatalf("failed to create http request, err = %v", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("http request failed: err = %v", err)
	}
	return resp
}

func setup(t *testing.T) client {
	svc := &mock.CommandService{}
	e := Endpoints{
		NewCommandEndpoint: MakeNewCommandEndpoint(svc),
	}
	h := MakeHTTPHandlers(
		context.Background(),
		e,
		httptransport.ServerErrorEncoder(EncodeError),
	)
	s := httptest.NewServer(h.NewCommandHandler)
	return client{s, svc, http.DefaultClient}
}
