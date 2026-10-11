package gin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type forkHookRequest struct {
	Name    string `json:"name" xml:"name" form:"name" yaml:"name" toml:"name" header:"X-Name" binding:"required"`
	calls   int
	context *Context
	hookErr error
}

func (r *forkHookRequest) AfterBind(c *Context) error {
	r.calls++
	r.context = c
	return r.hookErr
}

func TestForkAfterBindJSON(t *testing.T) {
	wantErr := errors.New("hook failure")
	var cases = []struct {
		name      string
		body      string
		hookErr   error
		wantCalls int
		wantErr   bool
	}{
		{name: "success", body: `{"name":"retained"}`, wantCalls: 1},
		{name: "hook error", body: `{"name":"retained"}`, hookErr: wantErr, wantCalls: 1, wantErr: true},
		{name: "validation error skips hook", body: `{}`, wantErr: true},
		{name: "decode error skips hook", body: `{`, wantErr: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			c, _ := CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			var request forkHookRequest
			request.hookErr = test.hookErr
			err := c.ShouldBindJSON2(&request)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, want error %v", err, test.wantErr)
			}
			if test.hookErr != nil && !errors.Is(err, test.hookErr) {
				t.Fatalf("hook error not preserved: %v", err)
			}
			if request.calls != test.wantCalls {
				t.Fatalf("hook calls = %d, want %d", request.calls, test.wantCalls)
			}
			if request.calls > 0 && request.context != c {
				t.Fatal("hook received a different context")
			}
		})
	}
}

func TestForkOriginalJSONDoesNotCallHook(t *testing.T) {
	c, _ := CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"retained"}`))
	var request forkHookRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		t.Fatal(err)
	}
	if request.calls != 0 {
		t.Fatalf("original API called hook %d times", request.calls)
	}
}

func TestForkJSONWithoutHook(t *testing.T) {
	c, _ := CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"retained"}`))
	var request struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON2(&request); err != nil {
		t.Fatal(err)
	}
	if request.Name != "retained" {
		t.Fatalf("name = %q", request.Name)
	}
}

func TestForkAfterBindWrappers(t *testing.T) {
	var cases = []struct {
		name string
		body string
		bind func(*Context, any) error
	}{
		{name: "XML", body: `<request><name>retained</name></request>`, bind: (*Context).ShouldBindXML2},
		{name: "Query", bind: (*Context).ShouldBindQuery2},
		{name: "YAML", body: "name: retained", bind: (*Context).ShouldBindYAML2},
		{name: "TOML", body: `name = "retained"`, bind: (*Context).ShouldBindTOML2},
		{name: "Header", bind: (*Context).ShouldBindHeader2},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			c, _ := CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/?name=retained", strings.NewReader(test.body))
			c.Request.Header.Set("X-Name", "retained")
			var request forkHookRequest
			if err := test.bind(c, &request); err != nil {
				t.Fatal(err)
			}
			if request.Name != "retained" || request.calls != 1 || request.context != c {
				t.Fatalf("unexpected bound value or hook state: %#v", request)
			}
		})
	}
}

func TestForkPlainWithoutHook(t *testing.T) {
	c, _ := CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("retained"))
	var request string
	if err := c.ShouldBindPlain2(&request); err != nil {
		t.Fatal(err)
	}
	if request != "retained" {
		t.Fatalf("plain value = %q", request)
	}
}
