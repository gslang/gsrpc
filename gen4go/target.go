package gen4go

import (
	"text/template"

	"github.com/gsdocker/gslang"
	"github.com/gsdocker/gslang/ast"
	"github.com/gsdocker/gslang/lexer"
	"github.com/gsdocker/gslogger"
)

var builtin = map[lexer.TokenType]string{
	lexer.KeySByte:   "int8",
	lexer.KeyByte:    "byte",
	lexer.KeyInt16:   "int16",
	lexer.KeyUInt16:  "uint16",
	lexer.KeyInt32:   "int32",
	lexer.KeyUInt32:  "uint32",
	lexer.KeyInt64:   "int64",
	lexer.KeyUInt64:  "uint64",
	lexer.KeyFloat32: "float32",
	lexer.KeyFloat64: "float64",
	lexer.KeyBool:    "bool",
	lexer.KeyString:  "string",
}

var defaultval = map[lexer.TokenType]string{
	lexer.KeySByte:   "int8(0)",
	lexer.KeyByte:    "byte(0)",
	lexer.KeyInt16:   "int16(0)",
	lexer.KeyUInt16:  "uint16(0)",
	lexer.KeyInt32:   "int32(0)",
	lexer.KeyUInt32:  "uint32(0)",
	lexer.KeyInt64:   "int64(0)",
	lexer.KeyUInt64:  "uint64(0)",
	lexer.KeyFloat32: "float32(0)",
	lexer.KeyFloat64: "float64(0)",
	lexer.KeyBool:    "false",
	lexer.KeyString:  "\"\"",
}

var imports = map[string]string{
	"gorpc.": "github.com/gsrpc/gorpc",
	"fmt.":   "fmt",
}

type _CodeTarget struct {
	gslogger.Log        // Log APIs
	rootpath     string // root path
}

// NewCodeTarget .
func NewCodeTarget(rootpath string) (gslang.CodeTarget, error) {

	codeGen := &_CodeTarget{
		Log:      gslogger.Get("gen4go"),
		rootpath: rootpath,
	}

	return codeGen, nil
}

func (target *_CodeTarget) Using() *template.Template {
	return nil
}
func (target *_CodeTarget) Table() *template.Template {
	return nil
}
func (target *_CodeTarget) Exception() *template.Template {
	return nil
}
func (target *_CodeTarget) Annotations() *template.Template {
	return nil
}
func (target *_CodeTarget) Enum() *template.Template {
	return nil
}
func (target *_CodeTarget) Contract() *template.Template {
	return nil
}

func (target *_CodeTarget) Begin() {

}

func (target *_CodeTarget) CreateScript(script *ast.Script, content []byte) {

}

func (target *_CodeTarget) End() {

}

// EndScript .
func (target *_CodeTarget) EndScript() {

	// content := codegen.content.String()
	//
	// for k, v := range imports {
	// 	if strings.Contains(content, k) {
	// 		codegen.header.WriteString(fmt.Sprintf("import \"%s\"\n", v))
	// 	}
	// }
	//
	// codegen.header.WriteString(content)
	//
	// //content, err := format.Source(codegen.header.Bytes())
	//
	// // if err != nil {
	// // 	gserrors.Panicf(err, "format golang source codes error")
	// // }
	//
	// fullpath := filepath.Join(codegen.rootpath, strings.Replace(codegen.script.Package, ".", "/", -1), filepath.Base(codegen.script.Name())+".go")
	//
	// codegen.I("generate golang file :%s", fullpath)
	//
	// if !fs.Exists(filepath.Dir(fullpath)) {
	// 	err := os.MkdirAll(filepath.Dir(fullpath), 0755)
	//
	// 	if err != nil {
	// 		gserrors.Panicf(err, "format golang source codes error")
	// 	}
	// }
	//
	// err := ioutil.WriteFile(fullpath, codegen.header.Bytes(), 0644)
	//
	// if err != nil {
	// 	gserrors.Panicf(err, "write generate golang file error")
	// }
}
