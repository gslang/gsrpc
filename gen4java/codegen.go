package gen4java

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
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
	lexer.KeyVoid:    "Void",
}

var builtinObj = map[lexer.TokenType]string{
	lexer.KeySByte:   "Byte",
	lexer.KeyByte:    "Byte",
	lexer.KeyInt16:   "Short",
	lexer.KeyUInt16:  "Short",
	lexer.KeyInt32:   "Integer",
	lexer.KeyUInt32:  "Integer",
	lexer.KeyInt64:   "Long",
	lexer.KeyUInt64:  "Long",
	lexer.KeyFloat32: "Float",
	lexer.KeyFloat64: "Double",
	lexer.KeyBool:    "Boolean",
	lexer.KeyString:  "String",
	lexer.KeyVoid:    "Void",
}

var readMapping = map[lexer.TokenType]string{
	lexer.KeySByte:   "reader.readSByte",
	lexer.KeyByte:    "reader.readByte",
	lexer.KeyInt16:   "reader.readInt16",
	lexer.KeyUInt16:  "reader.readUInt16",
	lexer.KeyInt32:   "reader.readInt32",
	lexer.KeyUInt32:  "reader.readUInt32",
	lexer.KeyInt64:   "reader.readInt64",
	lexer.KeyUInt64:  "reader.readUInt64",
	lexer.KeyFloat32: "reader.readFloat32",
	lexer.KeyFloat64: "reader.readFloat64",
	lexer.KeyBool:    "reader.readBoolean",
	lexer.KeyString:  "reader.readString",
}

var writeMapping = map[lexer.TokenType]string{
	lexer.KeySByte:   "writer.writeSByte",
	lexer.KeyByte:    "writer.writeByte",
	lexer.KeyInt16:   "writer.writeInt16",
	lexer.KeyUInt16:  "writer.writeUInt16",
	lexer.KeyInt32:   "writer.writeInt32",
	lexer.KeyUInt32:  "writer.writeUInt32",
	lexer.KeyInt64:   "writer.writeInt64",
	lexer.KeyUInt64:  "writer.writeUInt64",
	lexer.KeyFloat32: "writer.writeFloat32",
	lexer.KeyFloat64: "writer.writeFloat64",
	lexer.KeyBool:    "writer.writeBoolean",
	lexer.KeyString:  "writer.writeString",
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
	"Writer":     "import com.gsrpc.Writer",
	"Reader":     "import com.gsrpc.Reader",
	"ByteBuffer": "import java.nio.ByteBuffer",
}

type _CodeGen struct {
	gslogger.Log                    // Log APIs
	rootpath     string             // root path
	script       *ast.Script        // current script
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
		"exception": exception,
		"title":     strings.Title,
		"tableName": func(typeDecl ast.Type) string {
			if gslang.IsException(typeDecl) {
				return exception(strings.Title(typeDecl.Name()))
			}

			return strings.Title(typeDecl.Name())
		},
		"methodName":  methodName,
		"fieldName":   fieldname,
		"enumFields":  codeGen.enumFields,
		"notVoid":     gslang.NotVoid,
		"isPOD":       gslang.IsPOD,
		"isAsync":     gslang.IsAsync,
		"isException": gslang.IsException,
		"enumSize":    gslang.EnumSize,
		"enumType": func(typeDecl ast.Type) string {
			return builtin[gslang.EnumType(typeDecl)]
		},
		"builtin":         gslang.IsBuiltin,
		"typeName":        codeGen.typeName,
		"objTypeName":     codeGen.objTypeName,
		"defaultVal":      codeGen.defaultVal,
		"readType":        codeGen.readType,
		"writeType":       codeGen.writeType,
		"params":          codeGen.params,
		"returnParam":     codeGen.returnParam,
		"callArgs":        codeGen.callArgs,
		"returnArgs":      codeGen.returnArgs,
		"marshalField":    codeGen.marshalfield,
		"unmarshalField":  codeGen.unmarshalfield,
		"unmarshalParam":  codeGen.unmarshalParam,
		"methodcall":      codeGen.methodcall,
		"marshalParam":    codeGen.marshalParam,
		"marshalReturn":   codeGen.marshalReturn,
		"methodRPC":       codeGen.methodRPC,
		"marshalParams":   codeGen.marshalParams,
		"callback":        codeGen.callback,
		"unmarshalReturn": codeGen.unmarshalReturn,
		"constructor":     codeGen.constructor,
		"tagValue":        codeGen.tagValue,
	}

	tpl, err := template.New("t4java").Funcs(funcs).Parse(t4java)

	if err != nil {
		return nil, err
	}

	codeGen.tpl = tpl

	return codeGen, nil
}

func exception(name string) string {
	if strings.HasSuffix(name, "Exception") {
		return strings.Title(name)
	}

	return strings.Title(name) + "Exception"
}

func methodName(name string) string {
	return strings.ToLower(name[:1]) + name[1:]
}

func (codegen *_CodeGen) tagValue(typeDecl ast.Type) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)

		switch builtinType.Type {
		case lexer.KeySByte, lexer.KeyByte, lexer.KeyBool:
			return "com.gsrpc.Tag.I8.getValue()"
		case lexer.KeyInt16, lexer.KeyUInt16:
			return "com.gsrpc.Tag.I16.getValue()"
		case lexer.KeyInt32, lexer.KeyUInt32, lexer.KeyFloat32:
			return "com.gsrpc.Tag.I32.getValue()"
		case lexer.KeyInt64, lexer.KeyUInt64, lexer.KeyFloat64:
			return "com.gsrpc.Tag.I64.getValue()"
		case lexer.KeyString:
			return "com.gsrpc.Tag.String.getValue()"
		}

	case *ast.TypeRef:
		return codegen.tagValue(typeDecl.(*ast.TypeRef).Ref)
	case *ast.Enum:

		if codegen.enumSize(typeDecl) == 4 {
			return "com.gsrpc.Tag.I32.getValue()"
		}

		return "com.gsrpc.Tag.I8.getValue()"

	case *ast.Table:
		return "com.gsrpc.Tag.Table.getValue()"
	case *ast.Seq:

		seq := typeDecl.(*ast.Seq)

		component := codegen.tagValue(seq.Component)

		if component == "com.gsrpc.Tag.List.getValue()" {
			start, _ := gslang.Pos(typeDecl)
			gserrors.Panicf(nil, "list component %v can't be a list :%v", seq.Component, start)
		}

		return fmt.Sprintf("((%s << 4)|com.gsrpc.Tag.List.getValue())", component)
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", typeDecl)

	return ""
}

func (codegen *_CodeGen) callback(method *ast.Method) string {

	var buff bytes.Buffer

	buff.WriteString("future.Complete(null, ")

	if codegen.notVoid(method.Return) {
		buff.WriteString("returnParam, ")
	}

	buff.WriteString(");")

	return strings.Replace(buff.String(), ", );", ");", -1)
}

func (codegen *_CodeGen) constructor(fields []*ast.Field) string {
	var buff bytes.Buffer

	buff.WriteString("(")

	for _, field := range fields {
		buff.WriteString(fmt.Sprintf("%s %s, ", codegen.typeName(field.Type), fieldname(field.Name())))
	}

	buff.WriteString(")")

	return strings.Replace(buff.String(), ", )", " )", -1)
}

func (codegen *_CodeGen) marshalParams(params []*ast.Param) string {

	var buff bytes.Buffer

	for _, param := range params {

		buff.WriteString(codegen.marshalParam(param, fmt.Sprintf("arg%d", param.ID), 2))
	}

	return buff.String()
}

func (codegen *_CodeGen) methodRPC(method *ast.Method) string {

	var buff bytes.Buffer

	if codegen.notVoid(method.Return) {
		buff.WriteString(fmt.Sprintf("com.gsrpc.Future<%s> %s(", codegen.objTypeName(method.Return), methodName(method.Name())))
	} else {
		if gslang.IsAsync(method) {
			buff.WriteString(fmt.Sprintf("void %s(", methodName(method.Name())))
		} else {
			buff.WriteString(fmt.Sprintf("com.gsrpc.Future<Void> %s(", methodName(method.Name())))
		}

	}

	for _, v := range method.Params {
		buff.WriteString(fmt.Sprintf("%s arg%d, ", codegen.typeName(v.Type), v.ID))
	}

	if !gslang.IsAsync(method) {
		buff.WriteString("final int timeout)")
	} else {
		buff.WriteString(")")
	}

	return strings.Replace(buff.String(), ", )", ")", -1)
}

func (codegen *_CodeGen) marshalReturn(typeDecl ast.Type, varname string, indent int) string {
	var buff bytes.Buffer

	writeindent(&buff, indent)

	buff.WriteString("byte[] returnParam;\n\n")

	writeindent(&buff, indent)

	buff.WriteString("{\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString("com.gsrpc.BufferWriter writer = new com.gsrpc.BufferWriter();\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString(codegen.writeType(varname, typeDecl, indent+1))

	buff.WriteString("\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString("returnParam = writer.getContent();\n\n")

	writeindent(&buff, indent)

	buff.WriteString("}\n\n")
	return buff.String()
}

func (codegen *_CodeGen) marshalParam(param *ast.Param, varname string, indent int) string {
	var buff bytes.Buffer

	writeindent(&buff, indent)

	buff.WriteString("{\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString("com.gsrpc.BufferWriter writer = new com.gsrpc.BufferWriter();\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString(codegen.writeType(varname, param.Type, indent+1))

	buff.WriteString("\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString("com.gsrpc.Param param = new com.gsrpc.Param();\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString("param.setContent(writer.getContent());\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString(fmt.Sprintf("params[%d] = (param);\n\n", param.ID))

	writeindent(&buff, indent)

	buff.WriteString("}\n\n")
	return buff.String()
}

func (codegen *_CodeGen) unmarshalReturn(typeDecl ast.Type, varname string, ident int) string {
	var buff bytes.Buffer

	writeindent(&buff, ident)

	buff.WriteString(fmt.Sprintf("%s returnParam = %s;\n\n", codegen.typeName(typeDecl), codegen.defaultVal(typeDecl)))

	writeindent(&buff, ident)

	buff.WriteString("{\n\n")

	writeindent(&buff, ident+1)

	buff.WriteString(fmt.Sprintf("com.gsrpc.BufferReader reader = new com.gsrpc.BufferReader(%s.getContent());\n\n", varname))

	writeindent(&buff, ident+1)

	buff.WriteString(codegen.readType("returnParam", typeDecl, ident+1))

	buff.WriteString("\n\n")

	writeindent(&buff, ident)

	buff.WriteString("}\n\n")

	return buff.String()
}

func (codegen *_CodeGen) unmarshalParam(param *ast.Param, varname string, ident int) string {
	var buff bytes.Buffer

	writeindent(&buff, ident)

	buff.WriteString(fmt.Sprintf("%s arg%d = %s;\n\n", codegen.typeName(param.Type), param.ID, codegen.defaultVal(param.Type)))

	writeindent(&buff, ident)

	buff.WriteString("{\n\n")

	writeindent(&buff, ident+1)

	buff.WriteString(fmt.Sprintf("com.gsrpc.BufferReader reader = new com.gsrpc.BufferReader(%s.getParams()[%d].getContent());\n\n", varname, param.ID))

	writeindent(&buff, ident+1)

	buff.WriteString(codegen.readType(fmt.Sprintf("arg%d", param.ID), param.Type, ident+1))

	buff.WriteString("\n\n")

	writeindent(&buff, ident)

	buff.WriteString("}\n\n")

	return buff.String()
}

func (codegen *_CodeGen) methodcall(method *ast.Method) string {

	var buff bytes.Buffer

	if !codegen.notVoid(method.Return) {
		buff.WriteString(fmt.Sprintf("this.service.%s(", methodName(method.Name())))
	} else {
		buff.WriteString(fmt.Sprintf("%s ret = this.service.%s(", codegen.typeName(method.Return), methodName(method.Name())))
	}

	for i := range method.Params {
		buff.WriteString(fmt.Sprintf("arg%d, ", i))
	}

	buff.WriteString(");")

	return strings.Replace(buff.String(), ", );", ");", -1)
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
		buff.WriteString(fmt.Sprintf("%s %s, ", codegen.typeName(param.Type), param.Name()))
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
		return fmt.Sprintf("%s", codegen.typeName(param))
	}

	return "void"
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

	return prefix, strings.Title(nodes[len(nodes)-1])
}

func (codegen *_CodeGen) writeType(valname string, typeDecl ast.Type, indent int) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		return fmt.Sprintf("%s(%s);", writeMapping[builtinType.Type], valname)
	case *ast.TypeRef:
		typeRef := typeDecl.(*ast.TypeRef)

		return codegen.writeType(valname, typeRef.Ref, indent)

	case *ast.Enum, *ast.Table:
		return fmt.Sprintf("%s.marshal(writer);", valname)

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		isbytes := false

		builtinType, ok := seq.Component.(*ast.BuiltinType)

		if ok && builtinType.Type == lexer.KeyByte {
			isbytes = true
		}

		if seq.Size == -1 {

			if isbytes {

				return fmt.Sprintf("writer.writeBytes(%s);", valname)

			}

			var stream bytes.Buffer

			stream.WriteString(fmt.Sprintf("writer.writeUInt16((short)%s.length);\n\n", valname))

			writeindent(&stream, indent-1)

			stream.WriteString(fmt.Sprintf("for(%s v%d : %s){\n\n", codegen.typeName(seq.Component), indent, valname))

			writeindent(&stream, indent)

			stream.WriteString(codegen.writeType(fmt.Sprintf("v%d", indent), seq.Component, indent+1))

			stream.WriteString("\n\n")

			writeindent(&stream, indent-1)

			stream.WriteRune('}')

			return stream.String()

		}

		if isbytes {
			return fmt.Sprintf("writer.writeBytes(%s);", valname)
		}

		var stream bytes.Buffer

		stream.WriteString(fmt.Sprintf("writer.writeUInt16((short)%s.length);\n\n", valname))

		writeindent(&stream, indent-1)

		stream.WriteString(fmt.Sprintf("for(%s v%d : %s){\n\n", codegen.typeName(seq.Component), indent, valname))

		writeindent(&stream, indent)

		stream.WriteString(codegen.writeType(fmt.Sprintf("v%d", indent), seq.Component, indent+1))

		stream.WriteString("\n\n")

		writeindent(&stream, indent-1)

		stream.WriteRune('}')

		return stream.String()

	}

	gserrors.Panicf(nil, "writeType  error: unsupport type(%s)", typeDecl)

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

	case *ast.Enum:
		return fmt.Sprintf("%s = %s.unmarshal(reader);", valname, codegen.typeName(typeDecl))

	case *ast.Table:
		return fmt.Sprintf("%s.unmarshal(reader);", valname)

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		isbytes := false

		builtinType, ok := seq.Component.(*ast.BuiltinType)

		if ok && builtinType.Type == lexer.KeyByte {
			isbytes = true
		}

		if seq.Size == -1 {

			if isbytes {
				return fmt.Sprintf("%s = reader.readBytes();", valname)
			}

			var stream bytes.Buffer

			stream.WriteString(fmt.Sprintf("int max%d = reader.readUInt16();\n\n", indent))

			writeindent(&stream, indent-1)

			stream.WriteString(fmt.Sprintf("%s = new %s[max%d];\n\n", valname, codegen.arrayDefaultVal(seq.Component), indent))

			writeindent(&stream, indent-1)

			stream.WriteString(fmt.Sprintf("for(int i%d = 0; i%d < max%d; i%d ++ ){\n\n", indent, indent, indent, indent))

			writeindent(&stream, indent)

			stream.WriteString(fmt.Sprintf("%s v%d = %s;\n\n", codegen.typeName(seq.Component), indent, codegen.defaultVal(seq.Component)))

			writeindent(&stream, indent)

			stream.WriteString(codegen.readType(fmt.Sprintf("v%d", indent), seq.Component, indent+1))

			stream.WriteString("\n\n")

			writeindent(&stream, indent)

			stream.WriteString(fmt.Sprintf("%s[i%d] = v%d;\n\n", valname, indent, indent))

			writeindent(&stream, indent-1)

			stream.WriteRune('}')

			return stream.String()

		}

		if isbytes {
			return fmt.Sprintf("%s = reader.readBytes();", valname)
		}

		var stream bytes.Buffer

		stream.WriteString(fmt.Sprintf("for(int i%d = 0; i%d < %s.length; i%d ++ ){\n\n", indent, indent, valname, indent))

		writeindent(&stream, indent)

		stream.WriteString(fmt.Sprintf("%s v%d = %s[i%d];\n\n", codegen.typeName(seq.Component), indent, valname, indent))

		writeindent(&stream, indent)

		stream.WriteString(codegen.readType(fmt.Sprintf("v%d", indent), seq.Component, indent+1))

		stream.WriteString("\n\n")

		writeindent(&stream, indent)

		stream.WriteString(fmt.Sprintf("%s[i%d] = v%d;\n\n", valname, indent, indent))

		writeindent(&stream, indent-1)

		stream.WriteRune('}')

		return stream.String()
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", codegen.typeName)

	return "unknown"
}

func (codegen *_CodeGen) objTypeName(typeDecl ast.Type) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)

		return builtinObj[builtinType.Type]
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

		return fmt.Sprintf("%s[]", codegen.typeName(seq.Component))
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

	case *ast.Enum, *ast.Table:
		_, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())

		if gslang.IsException(typeDecl) {
			return exception(name)
		}

		return name

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		return fmt.Sprintf("%s[]", codegen.typeName(seq.Component))
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

		_, name := codegen.typeRef(typeDecl.Package(), typeDecl.FullName())
		//
		// if prefix != "" {
		// 	return prefix + "." + name + "." + enum.Constants[0].Name()
		// }

		return name + "." + enum.Constants[0].Name()

	case *ast.Table:

		return "new " + codegen.typeName(typeDecl) + "()"

	case *ast.Seq:
		return fmt.Sprintf("new %s", codegen.arrayDefaultVal(typeDecl))
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", typeDecl)

	return "unknown"
}

func (codegen *_CodeGen) arrayDefaultVal(typeDecl ast.Type) string {
	switch typeDecl.(type) {

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		return fmt.Sprintf("%s[0]", codegen.arrayDefaultVal(seq.Component))

	default:

		return codegen.typeName(typeDecl)

	}
}

func javaPackageName(origin string) string {

	return strings.Replace(origin, "/", ".", -1)
}

func (codegen *_CodeGen) writeJavaFile(name string, expr ast.Expr, content []byte) {

	var buff bytes.Buffer

	jPackageName := javaPackageName(codegen.packageName)

	buff.WriteString(fmt.Sprintf("package %s;\n\n", jPackageName))

	for _, i := range codegen.imports {

		buff.WriteString(fmt.Sprintf("%s;\n\n", i))

	}

	buff.Write(content)

	packagename := strings.Replace(jPackageName, ".", "/", -1)

	fullpath := filepath.Join(codegen.rootpath, packagename, name+".java")

	if err := os.MkdirAll(filepath.Dir(fullpath), 0755); err != nil {
		gserrors.Panicf(err, "create output directory error")
	}

	codegen.D("write file :%s", fullpath)

	if err := ioutil.WriteFile(fullpath, buff.Bytes(), 0644); err != nil {
		gserrors.Panicf(err, "write generate stub code error")
	}
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

	codegen.packageName = script.Package

	codegen.script = script

	codegen.imports = make(map[string]string)

	for k, v := range imports {
		codegen.imports[k] = v
	}

	return true
}

func (codegen *_CodeGen) Using(compiler *gslang.Compiler, using *ast.Using) {

	nodes := strings.Split(using.Name(), ".")

	_, ok := gslang.FindAnnotation(using.Ref, "gslang.Exception")

	name := using.Name()

	if ok {
		name = strings.Join(nodes[:len(nodes)-1], ".") + "." + exception(nodes[len(nodes)-1])
	}

	codegen.imports[nodes[len(nodes)-2]+"."] = "import " + name
}

func (codegen *_CodeGen) Table(compiler *gslang.Compiler, tableType *ast.Table) {

	var buff bytes.Buffer

	if err := codegen.tpl.ExecuteTemplate(&buff, "table", tableType); err != nil {
		gserrors.Panicf(err, "exec template(table) for %s error", tableType)
	}

	if gslang.IsException(tableType) {
		codegen.writeJavaFile(exception(tableType.Name()), tableType, buff.Bytes())
	} else {
		codegen.writeJavaFile(tableType.Name(), tableType, buff.Bytes())
	}

}

func (codegen *_CodeGen) Annotation(compiler *gslang.Compiler, annotation *ast.Table) {
}

func (codegen *_CodeGen) Enum(compiler *gslang.Compiler, enum *ast.Enum) {

	var buff bytes.Buffer

	if err := codegen.tpl.ExecuteTemplate(&buff, "enum", enum); err != nil {
		gserrors.Panicf(err, "exec template(enum) for %s error", enum)
	}

	codegen.writeJavaFile(enum.Name(), enum, buff.Bytes())
}
func (codegen *_CodeGen) Contract(compiler *gslang.Compiler, contract *ast.Contract) {

	var buff bytes.Buffer

	if err := codegen.tpl.ExecuteTemplate(&buff, "contract", contract); err != nil {
		gserrors.Panicf(err, "exec template(contract) for %s error", contract)
	}

	codegen.writeJavaFile(contract.Name(), contract, buff.Bytes())

	buff.Reset()

	if err := codegen.tpl.ExecuteTemplate(&buff, "dispatcher", contract); err != nil {
		gserrors.Panicf(err, "exec template(contract) for %s error", contract)
	}

	codegen.writeJavaFile(contract.Name()+"Dispatcher", contract, buff.Bytes())

	buff.Reset()

	if err := codegen.tpl.ExecuteTemplate(&buff, "rpc", contract); err != nil {
		gserrors.Panicf(err, "exec template(contract) for %s error", contract)
	}

	codegen.writeJavaFile(contract.Name()+"RPC", contract, buff.Bytes())
}

// EndScript .
func (codegen *_CodeGen) EndScript(compiler *gslang.Compiler) {

}
