package compiler

import ("os";"os/exec";"path/filepath";"strings";"testing")

func TestValidatorDecoratorsGeneratedDTO(t *testing.T) {
	dir := t.TempDir()
	src := `package validators

import "errors"

type CreateUserDTO struct {
 @IsEmail()
 @MinLength(5)
 Email string
 @IsCPF()
 CPF string
 @IsCNPJ()
 CNPJ string
 @Regex("^[a-z]{3}[0-9]{2}$")
 Code string
 @Min(18)
 @Max(120)
 Age int
 @IsOptional()
 @IsURL()
 Website *string
 @Validate(CheckName)
 Name string
}
func CheckName(v string) error {if v=="blocked" { return errors.New("blocked") };return nil}
`
	if err := os.WriteFile(filepath.Join(dir,"dto.mgo"),[]byte(src),0644);err!=nil {t.Fatal(err)}
	if err := CompileDir(dir);err!=nil {t.Fatal(err)}
	generated,err:=os.ReadFile(filepath.Join(dir,"dto_transformers_generated.go"))
	if err!=nil {t.Fatal(err)}
	for _,expected:=range []string{"Validate() error","__nestgo_validation.IsEmail(item)","__nestgo_validation.IsCPF(item)","__nestgo_validation.MatchRegexp(item","__nestgo_validation.ApplyFunc[string](item,CheckName)","__nestgo_regexp.MustCompile"} {
		if !strings.Contains(string(generated),expected) {t.Fatalf("missing %s in generated DTO: %s",expected,generated)}
	}
	root,err:=os.Getwd()
	if err!=nil {t.Fatal(err)}
	mod:="module validators\n\ngo 1.23\n\nrequire github.com/kevenmiano/nestgo v0.0.0\nreplace github.com/kevenmiano/nestgo => "+filepath.ToSlash(filepath.Dir(root))+"\n"
	if err:=os.WriteFile(filepath.Join(dir,"go.mod"),[]byte(mod),0644);err!=nil {t.Fatal(err)}
	test := `package validators
import("encoding/json";"testing";"github.com/kevenmiano/nestgo/validation")
func TestRules(t *testing.T) {
 var dto CreateUserDTO
 if err:=json.Unmarshal([]byte("{\"Email\":\"wrong\",\"CPF\":\"11111111111\",\"CNPJ\":\"00000000000000\",\"Code\":\"ABC\",\"Age\":17,\"Name\":\"blocked\"}"),&dto);err!=nil{t.Fatal(err)}
 err:=dto.Validate()
 issues,ok:=err.(validation.Errors)
 if !ok||len(issues)<5 {t.Fatalf("expected field errors: %v",err)}
 good:=CreateUserDTO{Email:"ada@example.com",CPF:"52998224725",CNPJ:"11.222.333/0001-81",Code:"abc12",Age:21,Name:"Ada"}
 if err:=good.Validate();err!=nil{t.Fatal(err)}
}
`
	if err:=os.WriteFile(filepath.Join(dir,"dto_test.go"),[]byte(test),0644);err!=nil {t.Fatal(err)}
	cmd:=exec.Command("go","test","-mod=mod",".")
	cmd.Dir=dir
	cmd.Env=append(os.Environ(),"GOWORK=off")
	out,err:=cmd.CombinedOutput()
	if err!=nil{t.Fatalf("generated DTO validation failed: %v\n%s",err,out)}
}
func TestValidatorDecoratorDiagnostics(t *testing.T) {
	tests:=[]struct{name,field,rule,want string}{
		{"wrong-type","Age int","@IsEmail()","requires string"},
		{"invalid-regex","Code string","@Regex(\"[\")","invalid regexp"},
		{"missing-regex","Code string","@Regex()","quoted regexp"},
		{"wrong-length","Name string","@MinLength(-1)","nonnegative"},
		{"length-reversed","Name string","@Length(5,2)","min <= max"},
		{"optional-nonpointer","Name string","@IsOptional()","pointer field"},
		{"excluded","Name string","@ExcludeInput()\n@IsEmail()","excluded"},
		{"constructor","Name string","@Validate(MakeRule())","function"},
	}
	for _,tc:=range tests {t.Run(tc.name,func(t *testing.T){
		source:="package demo\ntype DTO struct{\n"+tc.rule+"\n"+tc.field+"\n}\n"
		_,err:=Parse("dto.mgo",[]byte(source))
		if err==nil||!strings.Contains(err.Error(),tc.want){t.Fatalf("expected %s got %v",tc.want,err)}
	})}
}
