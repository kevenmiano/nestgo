package runtime

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/validation"
)

type validationResponseContext struct {
	status  int
	headers map[string]string
	body    []byte
}

func (c *validationResponseContext) Method() string       { return "POST" }
func (c *validationResponseContext) Path() string         { return "/create" }
func (c *validationResponseContext) Param(string) string  { return "" }
func (c *validationResponseContext) Query(string) string  { return "" }
func (c *validationResponseContext) Header(string) string { return "" }
func (c *validationResponseContext) Body() []byte         { return nil }
func (c *validationResponseContext) Status(code int)      { c.status = code }
func (c *validationResponseContext) SetHeader(key, value string) {
	if c.headers == nil {
		c.headers = map[string]string{}
	}
	c.headers[key] = value
}
func (c *validationResponseContext) Write(body []byte) error {
	c.body = append([]byte(nil), body...)
	return nil
}

func TestRespondValidationErrorDoesNotEchoInput(t *testing.T) {
	ctx := &validationResponseContext{}
	err := validation.Errors{{Field: "cpf", Rule: "IsCPF", Message: "invalid CPF"}}
	if e := respondValidationError(ctx, err); e != nil {
		t.Fatal(e)
	}
	if ctx.status != 400 || ctx.headers["Content-Type"] != "application/json" {
		t.Fatalf("status=%d headers=%v", ctx.status, ctx.headers)
	}
	var body struct {
		StatusCode int                     `json:"statusCode"`
		Fields     []validation.FieldError `json:"fields"`
	}
	if e := json.Unmarshal(ctx.body, &body); e != nil {
		t.Fatal(e)
	}
	if body.StatusCode != 400 || len(body.Fields) != 1 || body.Fields[0].Field != "cpf" {
		t.Fatalf("response: %s", ctx.body)
	}
	if strings.Contains(string(ctx.body), "52998224725") {
		t.Fatalf("sensitive input echoed: %s", ctx.body)
	}
}
