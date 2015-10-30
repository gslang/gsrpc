package gen4go

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"go/format"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsos/fs"
	"github.com/gsrpc/gslang"
	"github.com/gsrpc/gslang/ast"
	"github.com/gsrpc/gslang/lexer"
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

var readMapping = map[lexer.TokenType]string{
	lexer.KeySByte:   "gorpc.ReadSByte",
	lexer.KeyByte:    "gorpc.ReadByte",
	lexer.KeyInt16:   "gorpc.ReadInt16",
	lexer.KeyUInt16:  "gorpc.ReadUInt16",
	lexer.KeyInt32:   "gorpc.ReadInt32",
	lexer.KeyUInt32:  "gorpc.ReadUInt32",
	lexer.KeyInt64:   "gorpc.ReadInt64",
	lexer.KeyUInt64:  "gorpc.ReadUInt64",
	lexer.KeyFloat32: "gorpc.ReadFloat32",
	lexer.KeyFloat64: "gorpc.ReadFloat64",
	lexer.KeyBool:    "gorpc.ReadBool",
	lexer.KeyString:  "gorpc.ReadString",
}

var writeMapping = map[lexer.TokenType]string{
	lexer.KeySByte:   "gorpc.WriteSByte",
	lexer.KeyByte:    "gorpc.WriteByte",
	lexer.KeyInt16:   "gorpc.WriteInt16",
	lexer.KeyUInt16:  "gorpc.WriteUInt16",
	lexer.KeyInt32:   "gorpc.WriteInt32",
	lexer.KeyUInt32:  "gorpc.WriteUInt32",
	lexer.KeyInt64:   "gorpc.WriteInt64",
	lexer.KeyUInt64:  "gorpc.WriteUInt64",
	lexer.KeyFloat32: "gorpc.WriteFloat32",
	lexer.KeyFloat64: "gorpc.WriteFloat64",
	lexer.KeyBool:    "gorpc.WriteBool",
	lexer.KeyString:  "gorpc.WriteString",
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
	"gorpc.":    "github.com/gsrpc/gorpc",
	"trace.":    "github.com/gsrpc/gorpc/trace",
	"fmt.":      "fmt",
	"bytes.":    "bytes",
	"gsrpc.":    "com/gsrpc",
	"gserrors.": "github.com/gsdocker/gserrors",
}

type _CodeGen struct {
	gslogger.Log                    // Log APIs
	rootpath     string             // root path
	script       *ast.Script        // current script
	header       bytes.Buffer       // header writer
	content      bytes.Buffer       // content writer
	tpl          *template.Template // code generate template
	imports      map[string]string  // imports
	packageName  string             // package name
	scriptPath   string             // script path
	skips        []*regexp.Regexp   // skip lists
}

// NewCodeGen .
func NewCodeGen(rootpath string, skips []string) (gslang.Visitor, error) {

	codeGen := &_CodeGen{
		Log:      gslogger.Get("gen4go"),
		rootpath: rootpath,
	}

	for _, skip := range skips {
		exp, err := regexp.Compile(skip)

		if err != nil {
			return nil, gserrors.Newf(err, "invalid skip regex string :%s", skip)
		}

		codeGen.skips = append(codeGen.skips, exp)
	}

	funcs := template.FuncMap{
		"title": strings.Title,
		"enumType": func(typeDecl ast.Type) string {
			return builtin[gslang.EnumType(typeDecl)]
		},
		"notVoid":     gslang.NotVoid,
		"isPOD":       gslang.IsPOD,
		"isAsync":     gslang.IsAsync,
		"isException": gslang.IsException,
		"enumSize":    gslang.EnumSize,
		"builtin":     gslang.IsBuiltin,
		"typeName":    codeGen.typeName,
		"defaultVal":  codeGen.defaultVal,
		"readType":    codeGen.readType,
		"writeType":   codeGen.writeType,
		"params":      codeGen.params,
		"returnParam": codeGen.returnParam,
		"callArgs":    codeGen.callArgs,
		"returnArgs":  codeGen.returnArgs,
		"tagValue":    codeGen.tagValue,
	}

	tpl, err := template.New("gen4go").Funcs(funcs).Parse(tpl4go)

	if err != nil {
		return nil, err
	}

	codeGen.tpl = tpl

	return codeGen, nil
}

func (codegen *_CodeGen) tagValue(typeDecl ast.Type) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)

		switch builtinType.Type {
		case lexer.KeySByte, lexer.KeyByte, lexer.KeyBool:
			return "gorpc.TagI8"
		case lexer.KeyInt16, lexer.KeyUInt16:
			return "gorpc.TagI16"
		case lexer.KeyInt32, lexer.KeyUInt32, lexer.KeyFloat32:
			return "gorpc.TagI32"
		case lexer.KeyInt64, lexer.KeyUInt64, lexer.KeyFloat64:
			return "gorpc.TagI64"
		case lexer.KeyString:
			return "gorpc.TagString"
		}

	case *ast.TypeRef:
		return codegen.tagValue(typeDecl.(*ast.TypeRef).Ref)
	case *ast.Enum:

		if gslang.EnumSize(typeDecl) == 4 {
			return "gorpc.TagI32"
		}

		return "gorpc.TagI8"

	case *ast.Table:
		return "gorpc.TagTable"
	case *ast.Seq:

		seq := typeDecl.(*ast.Seq)

		component := codegen.tagValue(seq.Component)

		if component == "gorpc.TagList" {
			start, _ := gslang.Pos(typeDecl)
			gserrors.Panicf(nil, "list component %v can't be a list :%v", seq.Component, start)
		}

		return fmt.Sprintf("(%s << 4)|gorpc.TagList", component)
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", typeDecl)

	return ""
}

func (codegen *_CodeGen) params(params []*ast.Param) string {
	var buff bytes.Buffer

	buff.WriteString("(callSite *gorpc.CallSite, ")

	for _, param := range params {
		buff.WriteString(fmt.Sprintf("%s %s, ", param.Name(), codegen.typeName(param.Type)))
	}

	buff.WriteString(")")

	return strings.Replace(buff.String(), ", )", ")", 1)
}

func (codegen *_CodeGen) callArgs(params []*ast.Param) string {
	var buff bytes.Buffer

	buff.WriteString("(callSite, ")

	for _, param := range params {
		buff.WriteString(param.Name() + ", ")
	}

	buff.WriteString(")")

	return strings.Replace(buff.String(), ", )", ")", 1)
}

func (codegen *_CodeGen) returnParam(param ast.Type) string {
	if gslang.NotVoid(param) {
		return fmt.Sprintf("(retval %s,err error)", codegen.typeName(param))
	}

	return "(err error)"
}

func (codegen *_CodeGen) returnArgs(param ast.Type) string {
	if gslang.NotVoid(param) {
		return "retval,err"
	}

	return "err"
}

func (codegen *_CodeGen) typeRef(pacakgeName, fullname string) (prefix string, name string) {

	nodes := strings.Split(fullname, ".")

	if codegen.script.Package == pacakgeName {
		return "", strings.Title(nodes[len(nodes)-1])
	}

	prefix = nodes[len(nodes)-2]

	if strings.Join(nodes[:len(nodes)-1], ".") == "com.gsrpc" {
		prefix = "gorpc"
	}

	return prefix, strings.Title(nodes[len(nodes)-1])
}

func (codegen *_CodeGen) writeType(typeDecl ast.Type) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		return writeMapping[builtinType.Type]
	case *ast.TypeRef:
		typeRef := typeDecl.(*ast.TypeRef)

		return codegen.writeType(typeRef.Ref)

	case *ast.Enum:
		prefix, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if prefix != "" {
			return prefix + ".Write" + name
		}

		return "Write" + name

	case *ast.Table:

		prefix, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if prefix != "" {
			return "" + prefix + ".Write" + name
		}

		return "Write" + name

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		isbytes := false

		builtinType, ok := seq.Component.(*ast.BuiltinType)

		if ok && builtinType.Type == lexer.KeyByte {
			isbytes = true
		}

		var buff bytes.Buffer

		if seq.Size != -1 {

			if isbytes {

				if err := codegen.tpl.ExecuteTemplate(&buff, "writeByteArray", seq); err != nil {
					gserrors.Panicf(err, "exec template(writeByteArray) for %s errir", seq)
				}
			} else {

				if err := codegen.tpl.ExecuteTemplate(&buff, "writeArray", seq); err != nil {
					gserrors.Panicf(err, "exec template(writeArray) for %s errir", seq)
				}
			}

		} else {
			if isbytes {

				if err := codegen.tpl.ExecuteTemplate(&buff, "writeByteList", seq); err != nil {
					gserrors.Panicf(err, "exec template(writeByteList) for %s errir", seq)
				}
			} else {

				if err := codegen.tpl.ExecuteTemplate(&buff, "writeList", seq); err != nil {
					gserrors.Panicf(err, "exec template(writeList) for %s errir", seq)
				}
			}

		}

		return buff.String()
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", typeDecl)

	return "unknown"
}

func (codegen *_CodeGen) readType(typeDecl ast.Type) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		return readMapping[builtinType.Type]
	case *ast.TypeRef:
		typeRef := typeDecl.(*ast.TypeRef)

		return codegen.readType(typeRef.Ref)

	case *ast.Enum:
		prefix, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if prefix != "" {
			return prefix + ".Read" + name
		}

		return "Read" + name

	case *ast.Table:

		prefix, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if prefix != "" {
			return "" + prefix + ".Read" + name
		}

		return "Read" + name

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		var buff bytes.Buffer

		isbytes := false

		builtinType, ok := seq.Component.(*ast.BuiltinType)

		if ok && builtinType.Type == lexer.KeyByte {
			isbytes = true
		}

		if seq.Size != -1 {

			if isbytes {
				if err := codegen.tpl.ExecuteTemplate(&buff, "readByteArray", seq); err != nil {
					gserrors.Panicf(err, "exec template(readByteArray) for %s errir", seq)
				}
			} else {
				if err := codegen.tpl.ExecuteTemplate(&buff, "readArray", seq); err != nil {
					gserrors.Panicf(err, "exec template(readArray) for %s errir", seq)
				}
			}

		} else {

			if isbytes {
				if err := codegen.tpl.ExecuteTemplate(&buff, "readByteList", seq); err != nil {
					gserrors.Panicf(err, "exec template(readByteList) for %s errir", seq)
				}
			} else {
				if err := codegen.tpl.ExecuteTemplate(&buff, "readList", seq); err != nil {
					gserrors.Panicf(err, "exec template(readList) for %s errir", seq)
				}
			}

		}

		return buff.String()
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", typeDecl)

	return "unknown"
}

func (codegen *_CodeGen) typeName(typeDecl ast.Type) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		return builtin[builtinType.Type]
	case *ast.TypeRef:
		typeRef := typeDecl.(*ast.TypeRef)

		return codegen.typeName(typeRef.Ref)

	case *ast.Enum:
		prefix, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if prefix != "" {
			return prefix + "." + name
		}

		return name

	case *ast.Table:

		prefix, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if prefix != "" {
			return "*" + prefix + "." + name
		}

		return "*" + name

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		if seq.Size != -1 {
			return fmt.Sprintf("[%d]", seq.Size) + codegen.typeName(seq.Component)
		}

		return "[]" + codegen.typeName(seq.Component)
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", typeDecl)

	return "unknown"
}

func (codegen *_CodeGen) defaultVal(typeDecl ast.Type) string {

	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		return defaultval[builtinType.Type]
	case *ast.TypeRef:
		typeRef := typeDecl.(*ast.TypeRef)

		return codegen.defaultVal(typeRef.Ref)

	case *ast.Enum:

		enum := typeDecl.(*ast.Enum)

		prefix, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if prefix != "" {
			return prefix + "." + name + "" + enum.Constants[0].Name()
		}

		return name + "" + enum.Constants[0].Name()

	case *ast.Table:

		prefix, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if prefix != "" {
			return prefix + ".New" + name + "()"
		}

		return "New" + name + "()"

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		if seq.Size != -1 {

			var buff bytes.Buffer

			if err := codegen.tpl.ExecuteTemplate(&buff, "create_array", seq); err != nil {
				gserrors.Panicf(err, "exec template(create_array) for %s errir", seq)
			}

			return buff.String()
		}

		return "nil"
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", typeDecl)

	return "unknown"
}

func (codegen *_CodeGen) BeginScript(compiler *gslang.Compiler, script *ast.Script) bool {

	scriptPath := filepath.ToSlash(filepath.Clean(script.Name()))

	for _, skip := range codegen.skips {

		if skip.MatchString(scriptPath) {

			return false
		}
	}

	if strings.HasPrefix(script.Package, "gslang.") {
		return false
	}

	codegen.header.Reset()
	codegen.content.Reset()

	codegen.script = script

	codegen.packageName = script.Package
	codegen.scriptPath = strings.Replace(codegen.packageName, ".", "/", -1)

	langs := gslang.FindAnnotations(script.Module, "gslang.Package")

	for _, lang := range langs {

		langName, ok := lang.Args.NamedArg("Lang")

		if ok && compiler.Eval().EvalString(langName) == "golang" {

			packageName, ok := lang.Args.NamedArg("Name")

			if ok && compiler.Eval().EvalString(packageName) == script.Package {

				redirect, ok := lang.Args.NamedArg("Redirect")

				if ok {
					codegen.packageName = compiler.Eval().EvalString(redirect)
					codegen.scriptPath = codegen.packageName
				}
			}

		}
	}

	path := strings.Replace(codegen.packageName, ".", "/", -1)
	codegen.header.WriteString(fmt.Sprintf("package %s\n\n", filepath.Base(path)))

	codegen.imports = make(map[string]string)

	for k, v := range imports {
		codegen.imports[k] = v
	}

	return true
}

func (codegen *_CodeGen) Using(compiler *gslang.Compiler, using *ast.Using) {

	nodes := strings.Split(using.Name(), ".")

	codegen.imports[nodes[len(nodes)-2]+"."] = strings.Join(nodes[:len(nodes)-1], ".")
}

func (codegen *_CodeGen) Table(compiler *gslang.Compiler, tableType *ast.Table) {

	if err := codegen.tpl.ExecuteTemplate(&codegen.content, "table", tableType); err != nil {
		gserrors.Panicf(err, "exec template(table) for %s errir", tableType)
	}

}

func (codegen *_CodeGen) Annotation(compiler *gslang.Compiler, annotation *ast.Table) {
}

func (codegen *_CodeGen) Enum(compiler *gslang.Compiler, enum *ast.Enum) {
	if err := codegen.tpl.ExecuteTemplate(&codegen.content, "enum", enum); err != nil {
		gserrors.Panicf(err, "exec template(enum) for %s errir", enum)
	}
}

func (codegen *_CodeGen) Contract(compiler *gslang.Compiler, contract *ast.Contract) {
	if err := codegen.tpl.ExecuteTemplate(&codegen.content, "contract", contract); err != nil {
		gserrors.Panicf(err, "exec template(contract) for %s errir", contract)
	}
}

// EndScript .
func (codegen *_CodeGen) EndScript(compiler *gslang.Compiler) {

	content := codegen.content.String()

	packageName := codegen.script.Package

	if packageName == "com.gsrpc" {
		content = strings.Replace(content, "gorpc.", "", -1)
	}

	for k, v := range imports {
		if strings.Contains(content, k) {
			codegen.header.WriteString(fmt.Sprintf("import \"%s\"\n", v))
		}
	}

	codegen.header.WriteString(content)

	var err error
	var sources []byte

	fullpath := filepath.Join(codegen.rootpath, codegen.scriptPath, filepath.Base(codegen.script.Name())+".go")

	sources, err = format.Source(codegen.header.Bytes())

	if err != nil {
		gserrors.Panicf(err, "format golang source codes error:%s", fullpath)
	}

	codegen.D("generate golang file :%s", fullpath)

	if !fs.Exists(filepath.Dir(fullpath)) {
		err := os.MkdirAll(filepath.Dir(fullpath), 0755)

		if err != nil {
			gserrors.Panicf(err, "format golang source codes error")
		}
	}

	err = ioutil.WriteFile(fullpath, sources, 0644)

	if err != nil {
		gserrors.Panicf(err, "write generate golang file error")
	}
}
