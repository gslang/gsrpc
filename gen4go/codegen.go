package gen4go

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"go/format"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslang"
	"github.com/gsdocker/gslang/ast"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsos/fs"
)

// _CodeGen .
type _CodeGen struct {
	gslogger.Log                    // Log APIs
	rootpath     string             // root path
	script       *ast.Script        // current script
	header       bytes.Buffer       // bytes
	content      bytes.Buffer       // tables
	codebuilder  gslang.CodeBuilder // code builder
}

// NewCodeGen .
func NewCodeGen(rootpath string) (gslang.CodeGen, error) {

	codebuilder, err := gslang.NewCodeBuilder(tpl)

	if err != nil {
		return nil, err
	}

	return &_CodeGen{
		Log:         gslogger.Get("gen4go"),
		rootpath:    rootpath,
		codebuilder: codebuilder,
	}, nil
}

// BeginScript .
func (codegen *_CodeGen) BeginScript(script *ast.Script) {
	codegen.script = script
	codegen.header.Reset()
	codegen.content.Reset()
	codegen.codebuilder.Reset()

	packageName := filepath.Base(strings.Replace(script.Package, ".", "/", -1))

	codegen.header.WriteString(fmt.Sprintf("package %s\n\n", packageName))
}

// Using get using template
func (codegen *_CodeGen) Using(using *ast.Using) {

	if using.Ref.Package() == codegen.script.Package {
		return
	}

	codegen.codebuilder.CreateUsing(&codegen.header, fmt.Sprintf("import \"%s\"", strings.Replace(using.Ref.Package(), ".", "/", -1)))
}

func (codegen *_CodeGen) Table(table *ast.Table) {
	codegen.codebuilder.CreateTable(&codegen.content, table)
}

func (codegen *_CodeGen) Annotation(annotation *ast.Table) {
	// codegen.codebuilder.CreateAnnotation(&codegen.content, annotation)
}

func (codegen *_CodeGen) Enum(enum *ast.Enum) {
	codegen.codebuilder.CreateEnum(&codegen.content, enum)
}

func (codegen *_CodeGen) Contract(contract *ast.Contract) {
	codegen.codebuilder.CreateContract(&codegen.content, contract)
}

// EndScript .
func (codegen *_CodeGen) EndScript() {
	codegen.header.WriteString(codegen.content.String())

	_, err := format.Source(codegen.header.Bytes())

	if err != nil {
		gserrors.Panicf(err, "format golang source codes error")
	}

	fullpath := filepath.Join(codegen.rootpath, strings.Replace(codegen.script.Package, ".", "/", -1), filepath.Base(codegen.script.Name())+".go")

	codegen.I("generate golang file :%s", fullpath)

	if !fs.Exists(filepath.Dir(fullpath)) {
		err := os.MkdirAll(filepath.Dir(fullpath), 0600)

		if err != nil {
			gserrors.Panicf(err, "format golang source codes error")
		}
	}

	err = ioutil.WriteFile(fullpath, codegen.header.Bytes(), 0644)

	if err != nil {
		gserrors.Panicf(err, "write generate golang file error")
	}
}

// NewLine get new lines string
func (codegen *_CodeGen) NewLine() string {
	return "\n"
}
