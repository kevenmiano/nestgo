package compiler
import("os";"path/filepath";"strings";"testing")
func TestParseAndGenerate(t *testing.T){
 src:=[]byte(`package demo
@Controller("/users")
type UsersController struct{}
@Post("/")
@HttpCode(201)
func (c *UsersController) Create(@Body() dto string) string {return dto}
@Injectable()
type Service struct{}
@Module("users")
type Module struct{}
`)
 f,err:=Parse("users.mgo",src);if err!=nil{t.Fatal(err)}
 if strings.Contains(f.Source,"@Body")||strings.Contains(f.Source,"@Controller"){t.Fatalf("decorators not removed: %s",f.Source)}
 out,err:=Generate([]*File{f});if err!=nil{t.Fatal(err)}
 for _,want:=range []string{`Prefix: "/users"`,`Handler: "Create"`,`Status: 201`,`Kind: "Body"`,`Name: "users"`}{if !strings.Contains(string(out),want){t.Errorf("missing %q in registry: %s",want,out)}}
}
func TestInvalidDecorator(t *testing.T){_,err:=Parse("bad.mgo",[]byte("package demo\n@Unknown()\ntype A struct{}\n"));if err==nil||!strings.Contains(err.Error(),"unknown decorator"){t.Fatalf("unexpected error %v",err)}}
func TestOrphanRoute(t *testing.T){f,err:=Parse("bad.mgo",[]byte("package demo\ntype C struct{}\n@Get(\"/\")\nfunc (c *C) List() {}\n"));if err!=nil{t.Fatal(err)};_,err=Generate([]*File{f});if err==nil||!strings.Contains(err.Error(),"not decorated"){t.Fatalf("unexpected error %v",err)}}
func TestCompileDir(t *testing.T){dir:=t.TempDir();src:=[]byte("package demo\n@Controller(\"/\")\ntype C struct{}\n@Get(\"/\")\nfunc (c *C) List() string {return \"ok\"}\n");if err:=os.WriteFile(filepath.Join(dir,"a.mgo"),src,0644);err!=nil{t.Fatal(err)};if err:=CompileDir(dir);err!=nil{t.Fatal(err)};for _,name:=range []string{"a_generated.go","zz_modforge_metadata.go"}{if _,err:=os.Stat(filepath.Join(dir,name));err!=nil{t.Fatal(err)}}}
func TestLexerIgnoresAtInsideStringsAndComments(t *testing.T){src:=[]byte("package demo\n// @Unknown()\nvar text = \"@Unknown()\"\n@Controller(\"/users\")\ntype C struct{}\n@Get(\"/\")\nfunc (c *C) List() string {return text}\n");f,err:=Parse("x.mgo",src);if err!=nil{t.Fatal(err)};if !strings.Contains(f.Source,`"@Unknown()"`){t.Fatal("string modified")};if _,err=Generate([]*File{f});err!=nil{t.Fatal(err)}}
func TestLexerParamIndex(t *testing.T){src:=[]byte("package demo\n@Controller(\"/\")\ntype C struct{}\n@Post(\"/\")\nfunc (c *C) Create(id string, @Body() dto string) string {return dto}\n");f,err:=Parse("x.mgo",src);if err!=nil{t.Fatal(err)};if len(f.Declarations)!=2||len(f.Declarations[1].Params)!=1||f.Declarations[1].Params[0].Index!=1{t.Fatalf("unexpected params: %+v",f.Declarations)}}
