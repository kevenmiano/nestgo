package compiler

import ("fmt";"strconv";"strings";"unicode")
type marker struct {Offset int;Name,Argument string}
func lexDecorators(path string,src []byte)(string,map[int][]Annotation,[]marker,error){
 out:=append([]byte(nil),src...);annotations:=map[int][]Annotation{};params:=[]marker{};pending:=[]Annotation{};line:=1
 state:=byte(0);escaped:=false
 for i:=0;i<len(src);{
 c:=src[i]
 if c=='\n'{line++;if state==4{state=0};i++;continue}
 if state!=0{
 switch state{
 case 1,2:if escaped{escaped=false}else if c=='\\'{escaped=true}else if (state==1&&c=='"')||(state==2&&c=='\''){state=0}
 case 3:if c=='`'{state=0}
 case 5:if c=='*'&&i+1<len(src)&&src[i+1]=='/'{i+=2;state=0;continue}
 };i++;continue
 }
 if c=='/'&&i+1<len(src){if src[i+1]=='/'{state=4;i+=2;continue};if src[i+1]=='*'{state=5;i+=2;continue}}
 if c=='"'{state=1;i++;continue};if c=='\''{state=2;i++;continue};if c=='`'{state=3;i++;continue}
 if c=='@'&&i+1<len(src)&&(unicode.IsLetter(rune(src[i+1]))||src[i+1]=='_'){
 start:=i;startLine:=line;i++;begin:=i
 for i<len(src)&&(unicode.IsLetter(rune(src[i]))||unicode.IsDigit(rune(src[i]))||src[i]=='_'){i++}
 name:=string(src[begin:i]);if !supported[name]{return "",nil,nil,Diagnostic{path,startLine,"unknown decorator @"+name}}
 for i<len(src)&&(src[i]==' '||src[i]=='\t'){i++}
 argument:=""
 if i<len(src)&&src[i]=='('{
 i++;argStart:=i;depth:=1;quote:=byte(0);esc:=false
 for i<len(src)&&depth>0{
 ch:=src[i]
 if quote!=0{if esc{esc=false}else if ch=='\\'{esc=true}else if ch==quote{quote=0}}else if ch=='"'||ch=='\''||ch=='`'{quote=ch}else if ch=='('{depth++}else if ch==')'{depth--}
 if depth>0&&ch=='\n'{line++};i++
 }
 if depth!=0{return "",nil,nil,Diagnostic{path,startLine,"unclosed decorator argument"}}
 argument=strings.TrimSpace(string(src[argStart:i-1]))
 }
 for k:=start;k<i;k++{if out[k]!='\n'{out[k]=' '}}
 if name=="Body"||name=="Param"||name=="Query"||name=="Headers"{
 if argument!=""{v,e:=strconv.Unquote(argument);if e!=nil{return "",nil,nil,Diagnostic{path,startLine,"parameter decorator expects quoted string"}};argument=v}
 params=append(params,marker{start,name,argument})
 }else{
 lineStart:=start;for lineStart>0&&src[lineStart-1]!='\n'{lineStart--}
 lineEnd:=i;for lineEnd<len(src)&&src[lineEnd]!='\n'{lineEnd++}
 if strings.TrimSpace(string(src[lineStart:start]))!=""||strings.TrimSpace(string(src[i:lineEnd]))!=""{return "",nil,nil,Diagnostic{path,startLine,"declaration decorator must be on its own line"}}
 pending=append(pending,Annotation{name,argument,startLine})
 };continue
 }
 if len(pending)>0&&(c=='t'||c=='f')&&(i==0||src[i-1]=='\n'||src[i-1]==' '||src[i-1]=='\t'){
 if strings.HasPrefix(string(src[i:]),"type ")||strings.HasPrefix(string(src[i:]),"func ")||strings.HasPrefix(string(src[i:]),"type\t")||strings.HasPrefix(string(src[i:]),"func\t"){annotations[line]=append([]Annotation(nil),pending...);pending=nil}
 }
 i++
 }
 if len(pending)>0{return "",nil,nil,Diagnostic{path,pending[0].Line,"dangling decorator"}}
 if len(out)!=len(src){panic(fmt.Sprintf("offset drift %d != %d",len(out),len(src)))}
 return string(out),annotations,params,nil
}
