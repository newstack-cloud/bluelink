package corefunctions

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/function"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	. "gopkg.in/check.v1"
)

type HTTPResourceFunctionTestSuite struct {
	callStack   function.Stack
	callContext *functionCallContextMock
}

var _ = Suite(&HTTPResourceFunctionTestSuite{})

func (s *HTTPResourceFunctionTestSuite) SetUpTest(c *C) {
	s.callStack = function.NewStack()
	s.callContext = &functionCallContextMock{
		params: &core.ParamsImpl{},
		registry: &internal.FunctionRegistryMock{
			Functions: map[string]provider.Function{},
		},
		callStack: s.callStack,
	}
}

func (s *HTTPResourceFunctionTestSuite) Test_fetches_http_resource_successfully(c *C) {
	// Create a test HTTP server
	expectedData := []byte("test data content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, Equals, http.MethodGet)
		w.WriteHeader(http.StatusOK)
		w.Write(expectedData)
	}))
	defer server.Close()

	httpResourceFunc := NewHTTPResourceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "http_resource",
	})
	output, err := httpResourceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				server.URL,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, DeepEquals, expectedData)
}

func (s *HTTPResourceFunctionTestSuite) Test_fetches_json_resource(c *C) {
	// Create a test HTTP server with JSON content
	jsonData := []byte(`{"key": "value", "number": 42}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	}))
	defer server.Close()

	httpResourceFunc := NewHTTPResourceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "http_resource",
	})
	output, err := httpResourceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				server.URL,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, DeepEquals, jsonData)
}

func (s *HTTPResourceFunctionTestSuite) Test_fetches_binary_resource(c *C) {
	// Create a test HTTP server with binary content
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(binaryData)
	}))
	defer server.Close()

	httpResourceFunc := NewHTTPResourceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "http_resource",
	})
	output, err := httpResourceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				server.URL,
			},
			callCtx: s.callContext,
		},
		CallContext: s.callContext,
	})
	c.Assert(err, IsNil)
	c.Assert(output.ResponseData, DeepEquals, binaryData)
}

func (s *HTTPResourceFunctionTestSuite) Test_returns_error_for_404(c *C) {
	// Create a test HTTP server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	httpResourceFunc := NewHTTPResourceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "http_resource",
	})
	_, err := httpResourceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				server.URL,
			},
		},
		CallContext: s.callContext,
	})
	c.Assert(err, ErrorMatches, ".*failed with status code 404")
}

func (s *HTTPResourceFunctionTestSuite) Test_returns_error_for_500(c *C) {
	// Create a test HTTP server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	httpResourceFunc := NewHTTPResourceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "http_resource",
	})
	_, err := httpResourceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				server.URL,
			},
		},
		CallContext: s.callContext,
	})
	c.Assert(err, ErrorMatches, ".*failed with status code 500")
}

func (s *HTTPResourceFunctionTestSuite) Test_returns_error_for_invalid_url(c *C) {
	httpResourceFunc := NewHTTPResourceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "http_resource",
	})
	_, err := httpResourceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"not-a-valid-url://example.com",
			},
		},
		CallContext: s.callContext,
	})
	c.Assert(err, ErrorMatches, ".*failed to fetch resource.*")
}

func (s *HTTPResourceFunctionTestSuite) Test_returns_error_for_empty_url(c *C) {
	httpResourceFunc := NewHTTPResourceFunction()
	s.callStack.Push(&function.Call{
		FunctionName: "http_resource",
	})
	_, err := httpResourceFunc.Call(context.TODO(), &provider.FunctionCallInput{
		Arguments: &functionCallArgsMock{
			args: []any{
				"",
			},
		},
		CallContext: s.callContext,
	})
	c.Assert(err, ErrorMatches, "URL cannot be empty")
}
