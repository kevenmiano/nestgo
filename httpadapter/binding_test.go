package httpadapter

import (
	"errors"
	"testing"

	"github.com/kevenmiano/nestgo/validation"
)

type validationTestContext struct{ body []byte }
func (c validationTestContext) Method() string { return "POST" }
func (c validationTestContext) Path() string { return "/users" }
func (c validationTestContext) Param(string) string { return "" }
func (c validationTestContext) Query(string) string { return "" }
func (c validationTestContext) Header(string) string { return "" }
func (c validationTestContext) Body() []byte { return c.body }
func (c validationTestContext) Status(int) {}
func (c validationTestContext) SetHeader(string,string) {}
func (c validationTestContext) Write([]byte) error { return nil }

type validationTestDTO struct { Email string `json:"email"` }
func (u validationTestDTO) Validate() error {
	if !validation.IsEmail(u.Email) {
		return validation.Errors{{Field:"email",Rule:"IsEmail",Message:"invalid email"}}
	}
	return nil
}
func TestDecodeBodyRunsTypedDTOValidation(t *testing.T) {
	_,status,err:=DecodeBody[validationTestDTO](validationTestContext{body:[]byte(`{"email":"invalid"}`)})
	if status!=400 {t.Fatalf("status=%d, err=%v",status,err)}
	var fields validation.Errors
	if !errors.As(err,&fields) || len(fields)!=1 || fields[0].Field!="email" {
		t.Fatalf("unexpected field errors: %v",err)
	}
	dto,status,err:=DecodeBody[validationTestDTO](validationTestContext{body:[]byte(`{"email":"ada@example.com"}`)})
	if err!=nil || status!=0 || dto.Email!="ada@example.com" {
		t.Fatalf("valid DTO rejected: %+v status=%d err=%v",dto,status,err)
	}
}
