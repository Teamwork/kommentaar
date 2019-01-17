package test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamwork/utils/jsonutil"
	"github.com/teamwork/utils/stringutil"
)

// Code checks if the error code in the recoder matches the desired one, and
// will stop the test with t.Fatal() if it doesn't.
func Code(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("wrong response code\nwant: %d %s\ngot:  %d %s\nbody: %v",
			want, http.StatusText(want),
			recorder.Code, http.StatusText(recorder.Code),
			stringutil.Left(recorder.Body.String(), 150))
	}
}

// Default values for NewRequest()
var (
	DefaultHost        = "test.teamwork.dev"
	DefaultContentType = "application/json"
)

// NewRequest returns a new incoming server Request, suitable for passing to an
// echo.HandlerFunc for testing.
func NewRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)

	if req.Host == "" || req.Host == "example.com" {
		req.Host = DefaultHost
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", DefaultContentType)
	}

	return req
}

// Body returns the JSON representation of the passed in argument as an
// io.Reader. This is useful for creating a request body. For example:
//
//   NewRequest("POST", "/", echotest.Body(someStruct{
//       Foo: "bar",
//   }))
func Body(a interface{}) *bytes.Reader {
	return bytes.NewReader(jsonutil.MustMarshal(a))
}

// HTTP sets up a HTTP test. A GET request will be made for you if req is nil.
//
// For example:
//
//     rr := test.HTTP(t, nil, MyHandler)
//
// Or for a POST request:
//
//     req, err := http.NewRequest("POST", "/v1/email", b)
//     if err != nil {
//     	t.Fatalf("cannot make request: %v", err)
//     }
//     req.Header.Set("Content-Type", ct)
//     rr := test.HTTP(t, req, MyHandler)
func HTTP(t *testing.T, req *http.Request, h http.Handler) *httptest.ResponseRecorder {
	t.Helper()

	rr := httptest.NewRecorder()
	if req == nil {
		var err error
		req, err = http.NewRequest("GET", "", nil)
		if err != nil {
			t.Fatalf("cannot make request: %v", err)
		}
	}

	h.ServeHTTP(rr, req)
	return rr
}

// MultipartForm writes the keys and values from params to a multipart form.
//
// The first input parameter is used for "multipart/form-data" key/value
// strings, the optional second parameter is used creating file parts.
//
// Don't forget to set the Content-Type from the return value:
//
//   req.Header.Set("Content-Type", contentType)
func MultipartForm(params ...map[string]string) (b *bytes.Buffer, contentType string, err error) {
	b = &bytes.Buffer{}
	w := multipart.NewWriter(b)

	for k, v := range params[0] {
		field, err := w.CreateFormField(k)
		if err != nil {
			return nil, "", err
		}
		_, err = field.Write([]byte(v))
		if err != nil {
			return nil, "", err
		}
	}

	if len(params) > 1 {
		for k, v := range params[1] {
			field, err := w.CreateFormFile(k, k)
			if err != nil {
				return nil, "", err
			}
			_, err = field.Write([]byte(v))
			if err != nil {
				return nil, "", err
			}
		}
	}

	if err := w.Close(); err != nil {
		return nil, "", err
	}

	return b, w.FormDataContentType(), nil
}
