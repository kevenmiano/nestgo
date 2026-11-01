package sqlgen

import (
 "reflect"
 "strings"
 "testing"
)
func TestBindPostgres(t *testing.T) {
 q,args,err:=Bind("SELECT ':ignored', \"column:name\" FROM users WHERE id=:id AND alt=:id AND x=:name::text -- :comment\n/* :block */",map[string]any{"id":42,"name":"Ada"},PostgreSQL)
 if err!=nil {t.Fatal(err)}
 want:="SELECT ':ignored', \"column:name\" FROM users WHERE id=$1 AND alt=$2 AND x=$3::text -- :comment\n/* :block */"
 if q!=want {t.Fatalf("got %q want %q",q,want)}
 if !reflect.DeepEqual(args,[]any{42,42,"Ada"}) {t.Fatalf("args %#v",args)}
}
func TestQuestion(t *testing.T){
 q,args,err:=Bind("SELECT * FROM users WHERE id=:id",map[string]any{"id":7},Question)
 if err!=nil || q!="SELECT * FROM users WHERE id=?" || len(args)!=1 || args[0]!=7 {t.Fatalf("%q %#v %v",q,args,err)}
}
func TestValidation(t *testing.T){
 for _,tc:=range []struct{q string;p map[string]any}{
  {"SELECT :missing",nil},
  {"SELECT 1",map[string]any{"unused":1}},
  {"SELECT 'unterminated",nil},
  {"/* unterminated",nil},
 } {
  _,_,err:=Bind(tc.q,tc.p,Question)
  if err==nil {t.Fatalf("expected error for %q",tc.q)}
 }
 _,_,err:=Bind("SELECT 1",nil,Dialect(255))
 if err==nil || !strings.Contains(err.Error(),"unsupported") {t.Fatal(err)}
}
