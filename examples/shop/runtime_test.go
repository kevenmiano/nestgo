package shop
import("context";"github.com/kevenmiano/nestgo/runtime";"net/http/httptest";"strings";"testing")
func TestGeneratedRoutes(t *testing.T){
 app,err:=runtime.NewApp(context.Background(),runtime.WithRegistry("shop"));if err!=nil{t.Fatal(err)};defer app.Close();h:=app.HTTPHandler()
 for _,tc:=range []struct{method,path,body,want string;status int}{
 {"GET","/users/","","Ada",200},
 {"GET","/users/abc","","abc",200},
 {"POST","/users/",`{"name":"Grace"}`,"Grace",201},
 }{
 req:=httptest.NewRequest(tc.method,tc.path,strings.NewReader(tc.body));rec:=httptest.NewRecorder();h.ServeHTTP(rec,req)
 if rec.Code!=tc.status||!strings.Contains(rec.Body.String(),tc.want){t.Fatalf("%s %s: status=%d body=%s",tc.method,tc.path,rec.Code,rec.Body.String())}
 }
}
