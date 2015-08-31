package gen4java

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	lexer.KeySByte:   "byte",
	lexer.KeyByte:    "byte",
	lexer.KeyInt16:   "short",
	lexer.KeyUInt16:  "short",
	lexer.KeyInt32:   "int",
	lexer.KeyUInt32:  "int",
	lexer.KeyInt64:   "long",
	lexer.KeyUInt64:  "long",
	lexer.KeyFloat32: "float",
	lexer.KeyFloat64: "double",
	lexer.KeyBool:    "boolean",
	lexer.KeyString:  "String",
}

var readMapping = map[lexer.TokenType]string{
	lexer.KeySByte:   "reader.ReadSByte",
	lexer.KeyByte:    "reader.ReadByte",
	lexer.KeyInt16:   "reader.ReadInt16",
	lexer.KeyUInt16:  "reader.ReadUInt16",
	lexer.KeyInt32:   "reader.ReadInt32",
	lexer.KeyUInt32:  "reader.ReadUInt32",
	lexer.KeyInt64:   "reader.ReadInt64",
	lexer.KeyUInt64:  "reader.ReadUInt6",
	lexer.KeyFloat32: "reader.ReadFloat32",
	lexer.KeyFloat64: "reader.ReadFloat64",
	lexer.KeyBool:    "reader.ReadBool",
	lexer.KeyString:  "reader.ReadString",
}

var writeMapping = map[lexer.TokenType]string{
	lexer.KeySByte:   "writer.WriteSByte",
	lexer.KeyByte:    "writer.WriteByte",
	lexer.KeyInt16:   "writer.WriteInt16",
	lexer.KeyUInt16:  "writer.WriteUInt16",
	lexer.KeyInt32:   "writer.WriteInt32",
	lexer.KeyUInt32:  "writer.WriteUInt32",
	lexer.KeyInt64:   "writer.WriteInt64",
	lexer.KeyUInt64:  "writer.WriteUInt6",
	lexer.KeyFloat32: "writer.WriteFloat32",
	lexer.KeyFloat64: "writer.WriteFloat64",
	lexer.KeyBool:    "writer.WriteBool",
	lexer.KeyString:  "writer.WriteString",
}

var defaultval = map[lexer.TokenType]string{
	lexer.KeySByte:   "0",
	lexer.KeyByte:    "0",
	lexer.KeyInt16:   "0",
	lexer.KeyUInt16:  "0",
	lexer.KeyInt32:   "0",
	lexer.KeyUInt32:  "0",
	lexer.KeyInt64:   "0",
	lexer.KeyUInt64:  "0",
	lexer.KeyFloat32: "0",
	lexer.KeyFloat64: "0",
	lexer.KeyBool:    "false",
	lexer.KeyString:  "\"\"",
}

var imports = map[string]string{
	"gorpc.":    "github.com/gsrpc/gorpc",
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
}

// NewCodeGen .
func NewCodeGen(rootpath string) (gslang.Visitor, error) {

	codeGen := &_CodeGen{
		Log:      gslogger.Get("gen4go"),
		rootpath: rootpath,
	}

	funcs := template.FuncMap{
		"title":          strings.Title,
		"fieldName":      fieldname,
		"enumFields":     codeGen.enumFields,
		"enumType":       codeGen.enumType,
		"enumSize":       codeGen.enumSize,
		"typeName":       codeGen.typeName,
		"defaultVal":     codeGen.defaultVal,
		"builtin":        codeGen.builtin,
		"readType":       codeGen.readType,
		"writeType":      codeGen.writeType,
		"params":         codeGen.params,
		"returnParam":    codeGen.returnParam,
		"callArgs":       codeGen.callArgs,
		"returnArgs":     codeGen.returnArgs,
		"notVoid":        codeGen.notVoid,
		"marshalField":   codeGen.marshalfield,
		"unmarshalField": codeGen.unmarshalfield,
	}

	tpl, err := template.New("t4java").Funcs(funcs).Parse(t4java)

	if err != nil {
		return nil, err
	}

	codeGen.tpl = tpl

	return codeGen, nil
}

func fieldname(name string) string {
	return strings.ToLower(string(name[0])) + name[1:]
}

func writeindent(stream *bytes.Buffer, indent int) {
	for i := 0; i < indent; i++ {
		stream.WriteRune('\t')
	}
}

func (codegen *_CodeGen) marshalfield(field *ast.Field) string {
	return codegen.writeType(fieldname(field.Name()), field.Type, 3)
}

func (codegen *_CodeGen) unmarshalfield(field *ast.Field) string {
	return codegen.readType(fieldname(field.Name()), field.Type, 3)
}

func (codegen *_CodeGen) notVoid(typeDecl ast.Type) bool {
	builtinType, ok := typeDecl.(*ast.BuiltinType)

	if !ok {
		return true
	}

	return builtinType.Type != lexer.KeyVoid
}

func (codegen *_CodeGen) enumFields(enum *ast.Enum) string {
	var buff bytes.Buffer

	for _, constant := range enum.Constants {
		buff.WriteString(fmt.Sprintf("%s((%s)%d),\n\t", strings.Title(constant.Name()), codegen.enumType(enum), constant.Value))
	}

	content := buff.String()

	return content[:len(content)-3] + ";"
}

func (codegen *_CodeGen) enumType(typeDecl ast.Type) string {
	_, ok := gslang.FindAnnotation(typeDecl, "gslang.Flag")

	if ok {
		return builtin[lexer.KeyUInt32]
	}

	return builtin[lexer.KeyByte]
}

func (codegen *_CodeGen) builtin(typeDecl ast.Type) bool {
	_, ok := typeDecl.(*ast.BuiltinType)

	return ok
}

func (codegen *_CodeGen) enumSize(typeDecl ast.Type) int {
	_, ok := gslang.FindAnnotation(typeDecl, "gslang.Flag")

	if ok {
		return 4
	}

	return 1
}

func (codegen *_CodeGen) params(params []*ast.Param) string {
	var buff bytes.Buffer

	buff.WriteString("(")

	for _, param := range params {
		buff.WriteString(fmt.Sprintf("%s %s, ", param.Name(), codegen.typeName(param.Type)))
	}

	buff.WriteString(")")

	return strings.Replace(buff.String(), ", )", ")", 1)
}

func (codegen *_CodeGen) callArgs(params []*ast.Param) string {
	var buff bytes.Buffer

	buff.WriteString("(")

	for _, param := range params {
		buff.WriteString(param.Name() + ", ")
	}

	buff.WriteString(")")

	return strings.Replace(buff.String(), ", )", ")", 1)
}

func (codegen *_CodeGen) returnParam(param ast.Type) string {
	if codegen.notVoid(param) {
		return fmt.Sprintf("(retval %s,err error)", codegen.typeName(param))
	}

	return "(err error)"
}

func (codegen *_CodeGen) returnArgs(param ast.Type) string {
	if codegen.notVoid(param) {
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

func (codegen *_CodeGen) writeType(valname string, typeDecl ast.Type, indent int) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		return fmt.Sprintf("%s(%s)", writeMapping[builtinType.Type], valname)
	case *ast.TypeRef:
		typeRef := typeDecl.(*ast.TypeRef)

		return codegen.writeType(valname, typeRef.Ref, indent)

	case *ast.Enum, *ast.Table:
		return fmt.Sprintf("%s.Marshal(writer);", valname)

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		isbytes := false

		builtinType, ok := seq.Component.(*ast.BuiltinType)

		if ok && builtinType.Type == lexer.KeyByte {
			isbytes = true
		}

		if seq.Size == -1 {

			if isbytes {

				return fmt.Sprintf("writer.WriteBytes(%s);", valname)

			}

			var stream bytes.Buffer

			stream.WriteString(fmt.Sprintf("writer.WriteUint16((short)%s.length);\n", valname))

			writeindent(&stream, indent)

			stream.WriteString(fmt.Sprintf("for(%s v%d : %s){\n", codegen.typeName(seq.Component), indent, valname))

			stream.WriteString(codegen.writeType(fmt.Sprintf("v%d", indent), seq.Component, indent+1))

			writeindent(&stream, indent)

			stream.WriteRune('}')

			return stream.String()

		}

		if isbytes {
			return fmt.Sprintf("writer.WriteseqBytes(%s)", valname)
		}

		var stream bytes.Buffer

		stream.WriteString(fmt.Sprintf("writer.WriteUint16((short)%s.length);\n", valname))

		stream.WriteString(fmt.Sprintf("for(%s v%d : %s){\n", codegen.typeName(seq.Component), indent, valname))

		stream.WriteString(codegen.writeType(fmt.Sprintf("v%d", indent), seq.Component, indent+1))

		writeindent(&stream, indent)

		stream.WriteRune('}')

	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", codegen.typeName)

	return "unknown"
}

func (codegen *_CodeGen) readType(valname string, typeDecl ast.Type, indent int) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		return fmt.Sprintf("%s = %s();", valname, readMapping[builtinType.Type])
	case *ast.TypeRef:
		typeRef := typeDecl.(*ast.TypeRef)

		return codegen.readType(valname, typeRef.Ref, indent)

	case *ast.Enum, *ast.Table:

		prefix, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if prefix != "" {
			return fmt.Sprintf("%s = %s.%s.Unmarshal(reader);", valname, prefix, name)
		}

		return fmt.Sprintf("%s = %s.Unmarshal(reader);", valname, name)

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		isbytes := false

		builtinType, ok := seq.Component.(*ast.BuiltinType)

		if ok && builtinType.Type == lexer.KeyByte {
			isbytes = true
		}

		if seq.Size == -1 {

			if isbytes {
				return fmt.Sprintf("%s = reader.ReadBytes();", valname)
			}

			var stream bytes.Buffer

			stream.WriteString(fmt.Sprintf("int imax%d = reader.ReadUint16();\n\n", indent))

			writeindent(&stream, indent)

			stream.WriteString(fmt.Sprintf("%s = new %s[imax%d];\n\n", valname, codegen.typeName(seq.Component), indent))

			writeindent(&stream, indent)

			stream.WriteString(fmt.Sprintf("for(int i%d = 0; i%d < imax%d; i%d ++ ){\n\n", indent, indent, indent, indent))

			writeindent(&stream, indent+1)

			stream.WriteString(fmt.Sprintf("%s v%d = %s;\n\n", codegen.typeName(seq.Component), indent, codegen.defaultVal(seq.Component)))

			stream.WriteString(codegen.readType(fmt.Sprintf("v%d", indent), seq.Component, indent+1))

			stream.WriteRune('\n')

			writeindent(&stream, indent+1)

			stream.WriteString(fmt.Sprintf("%s[i%d] = v%d;\n\n", valname, indent, indent))

			writeindent(&stream, indent)

			stream.WriteRune('}')

			return stream.String()

		}

		if isbytes {
			return fmt.Sprintf("reader.ReadseqBytes(%s);", valname)
		}

		var stream bytes.Buffer

		stream.WriteString(fmt.Sprintf("int imax%d = reader.ReadUint16();\n\n", indent))

		writeindent(&stream, indent)

		stream.WriteString(fmt.Sprintf("%s = new %s[imax%d];\n\n", valname, codegen.typeName(seq.Component), indent))

		writeindent(&stream, indent)

		stream.WriteString(fmt.Sprintf("for(int i%d = 0; i%d < imax%d; i%d ++ ){\n\n", indent, indent, indent, indent))

		writeindent(&stream, indent+1)

		stream.WriteString(fmt.Sprintf("%s v%d = %s;\n\n", codegen.typeName(seq.Component), indent, codegen.defaultVal(seq.Component)))

		stream.WriteString(codegen.readType(fmt.Sprintf("v%d", indent), seq.Component, indent+1))

		stream.WriteRune('\n')

		writeindent(&stream, indent+1)

		stream.WriteString(fmt.Sprintf("%s[i%d] = v%d;\n\n", valname, indent, indent))

		writeindent(&stream, indent)

		stream.WriteRune('}')

		return stream.String()
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", codegen.typeName)

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

	case *ast.Enum, *ast.Table:
		prefix, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if prefix != "" {
			return prefix + "." + name
		}

		return name

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		if seq.Size != -1 {
			return fmt.Sprintf("%s[%d]", codegen.typeName(seq.Component), seq.Size)
		}

		return fmt.Sprintf("seqseq<%s>", codegen.typeName(seq.Component))
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", codegen.typeName)

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

			if err := codegen.tpl.ExecuteTemplate(&buff, "create_seq", seq); err != nil {
				gserrors.Panicf(err, "exec template(create_seq) for %s errir", seq)
			}

			return buff.String()
		}

		return "nil"
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", codegen.typeName)

	return "unknown"
}

func (codegen *_CodeGen) BeginScript(compiler *gslang.Compiler, script *ast.Script) {

	codegen.header.Reset()
	codegen.content.Reset()

	codegen.script = script

	codegen.packageName = script.Package
	codegen.scriptPath = strings.Replace(codegen.packageName, ".", "/", -1)

	lang, ok := gslang.FindAnnotation(script, "gslang.Lang")

	if ok {

		langName, ok := lang.Args.NamedArg("Name")

		if ok {
			if compiler.Eval().EvalString(langName) == "golang" {
				packageName, ok := lang.Args.NamedArg("Package")
				if ok {
					codegen.packageName = compiler.Eval().EvalString(packageName)
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
func (codegen *_CodeGen) Exception(compiler *gslang.Compiler, tableType *ast.Table) {
	if err := codegen.tpl.ExecuteTemplate(&codegen.content, "exception", tableType); err != nil {
		gserrors.Panicf(err, "exec template(exception) for %s errir", tableType)
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
