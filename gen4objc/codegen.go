package gen4objc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
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
	lexer.KeySByte:   "SInt8",
	lexer.KeyByte:    "UInt8",
	lexer.KeyInt16:   "SInt16",
	lexer.KeyUInt16:  "UInt16",
	lexer.KeyInt32:   "SInt32",
	lexer.KeyUInt32:  "UInt32",
	lexer.KeyInt64:   "SInt64",
	lexer.KeyUInt64:  "UInt64",
	lexer.KeyFloat32: "Float32",
	lexer.KeyFloat64: "Float64",
	lexer.KeyBool:    "BOOL",
	lexer.KeyString:  "NSString*",
	lexer.KeyVoid:    "void",
}

var readMapping = map[lexer.TokenType]string{
	lexer.KeySByte:   "reader ReadSByte",
	lexer.KeyByte:    "reader ReadByte",
	lexer.KeyInt16:   "reader ReadInt16",
	lexer.KeyUInt16:  "reader ReadUInt16",
	lexer.KeyInt32:   "reader ReadInt32",
	lexer.KeyUInt32:  "reader ReadUInt32",
	lexer.KeyInt64:   "reader ReadInt64",
	lexer.KeyUInt64:  "reader ReadUInt64",
	lexer.KeyFloat32: "reader ReadFloat32",
	lexer.KeyFloat64: "reader ReadFloat64",
	lexer.KeyBool:    "reader ReadBool",
	lexer.KeyString:  "reader ReadString",
}

var writeMapping = map[lexer.TokenType]string{
	lexer.KeySByte:   "writer WriteSByte",
	lexer.KeyByte:    "writer WriteByte",
	lexer.KeyInt16:   "writer WriteInt16",
	lexer.KeyUInt16:  "writer WriteUInt16",
	lexer.KeyInt32:   "writer WriteInt32",
	lexer.KeyUInt32:  "writer WriteUInt32",
	lexer.KeyInt64:   "writer WriteInt64",
	lexer.KeyUInt64:  "writer WriteUInt64",
	lexer.KeyFloat32: "writer WriteFloat32",
	lexer.KeyFloat64: "writer WriteFloat64",
	lexer.KeyBool:    "writer WriteBool",
	lexer.KeyString:  "writer WriteString",
}

var defaultval = map[lexer.TokenType]string{
	lexer.KeySByte:   "(SInt8)0",
	lexer.KeyByte:    "(UInt8)0",
	lexer.KeyInt16:   "(SInt16)0",
	lexer.KeyUInt16:  "(UInt16)0",
	lexer.KeyInt32:   "(SInt32)0",
	lexer.KeyUInt32:  "(UInt32)0",
	lexer.KeyInt64:   "(Int64)0",
	lexer.KeyUInt64:  "(UInt64)0",
	lexer.KeyFloat32: "(Float32)0",
	lexer.KeyFloat64: "(Float64)0",
	lexer.KeyBool:    "FALSE",
	lexer.KeyString:  "@\"\"",
}

var id2Type = map[lexer.TokenType]string{
	lexer.KeySByte:   "(SInt8)((NSNumber*)%s).charValue",
	lexer.KeyByte:    "(UInt8)((NSNumber*)%s).unsignedCharValue",
	lexer.KeyInt16:   "(SInt16)((NSNumber*)%s).shortValue",
	lexer.KeyUInt16:  "(UInt16)((NSNumber*)%s).unsignedShortValue",
	lexer.KeyInt32:   "(SInt32)((NSNumber*)%s).intValue",
	lexer.KeyUInt32:  "(UInt32)((NSNumber*)%s).unsignedIntValue",
	lexer.KeyInt64:   "(Int64)((NSNumber*)%s).longLongValue",
	lexer.KeyUInt64:  "(UInt64)((NSNumber*)%s).unsignedLongLongValue",
	lexer.KeyFloat32: "(Float32)((NSNumber*)%s).floatValue",
	lexer.KeyFloat64: "(Float64)((NSNumber*)%s).doubleValue",
	lexer.KeyBool:    "(BOOL)((NSNumber*)%s).boolValue",
	lexer.KeyString:  "(NSString*)%s",
}

var type2id = map[lexer.TokenType]string{
	lexer.KeySByte:   "[[NSNumber alloc]initWithChat :%s]",
	lexer.KeyByte:    "[[NSNumber alloc] initWithUnsignedChar:%s]",
	lexer.KeyInt16:   "[[NSNumber alloc] initWithShort:%s]",
	lexer.KeyUInt16:  "[[NSNumber alloc] initWithUnsignedShort:%s]",
	lexer.KeyInt32:   "[[NSNumber alloc] initWithInt:%s]",
	lexer.KeyUInt32:  "[[NSNumber alloc] initWithUnsignedInt:%s]",
	lexer.KeyInt64:   "[[NSNumber alloc] initWithLongLong:%s]",
	lexer.KeyUInt64:  "[[NSNumber alloc] initWithUnsignedLongLong:%s]",
	lexer.KeyFloat32: "[[NSNumber alloc] initWithFloat:%s]",
	lexer.KeyFloat64: "[[NSNumber alloc] initWithDouble:%s]",
	lexer.KeyBool:    "[[NSNumber alloc] initWithBool:%s]",
	lexer.KeyString:  "(id)%s",
}

var imports = map[string]string{
	"GSWriter":     "#import <com/gsrpc/stream.h>",
	"GSReader":     "#import <com/gsrpc/stream.h>",
	"GSDispatcher": "#import <com/gsrpc/channel.h>",
	"GSChannel":    "#import <com/gsrpc/channel.h>",
}

type _CodeGen struct {
	gslogger.Log                    // Log APIs
	rootpath     string             // root path
	script       *ast.Script        // current script
	tpl          *template.Template // code generate template
	imports      map[string]string  // imports
	prefix       map[string]string  // package name
	predecl      bytes.Buffer       //header file writer buffer
	header       bytes.Buffer       //header file writer buffer
	source       bytes.Buffer       //header file writer buffer
	compiler     *gslang.Compiler   // compilers
	skips        []*regexp.Regexp   // skip lists
}

// NewCodeGen .
func NewCodeGen(rootpath string, skips []string) (gslang.Visitor, error) {

	codeGen := &_CodeGen{
		Log:      gslogger.Get("gen4go"),
		rootpath: rootpath,
		prefix:   make(map[string]string),
	}

	for _, skip := range skips {
		exp, err := regexp.Compile(skip)

		if err != nil {
			return nil, gserrors.Newf(err, "invalid skip regex string :%s", skip)
		}

		codeGen.skips = append(codeGen.skips, exp)
	}

	funcs := template.FuncMap{
		"title":           codeGen.title,
		"title2":          strings.Title,
		"enumFields":      codeGen.enumFields,
		"typeName":        codeGen.typeName,
		"enumRead":        codeGen.enumRead,
		"enumWrite":       codeGen.enumWrite,
		"fieldDecl":       codeGen.fieldDecl,
		"defaultVal":      codeGen.defaultVal,
		"marshalField":    codeGen.marshalField,
		"unmarshalField":  codeGen.unmarshalField,
		"methodDecl":      codeGen.methodDecl,
		"rpcMethodDecl":   codeGen.rpcMethodDecl,
		"unmarshalParam":  codeGen.unmarshalParam,
		"marshalParam":    codeGen.marshalParam,
		"methodCall":      codeGen.methodCall,
		"marshalReturn":   codeGen.marshalReturn,
		"unmarshalReturn": codeGen.unmarshalReturn,
		"notVoid":         gslang.NotVoid,
		"isPOD":           gslang.IsPOD,
		"isAsync":         gslang.IsAsync,
		"isException":     gslang.IsException,
		"enumSize":        gslang.EnumSize,
		"enumType": func(typeDecl ast.Type) string {
			return builtin[gslang.EnumType(typeDecl)]
		},
		"builtin":       gslang.IsBuiltin,
		"marshalParams": codeGen.marshalParams,
		"callback":      codeGen.callback,
		"tagValue":      codeGen.tagValue,
	}

	tpl, err := template.New("t4objc").Funcs(funcs).Parse(t4objc)

	if err != nil {
		return nil, err
	}

	codeGen.tpl = tpl

	return codeGen, nil
}

func (codegen *_CodeGen) callback(method *ast.Method) string {

	var buff bytes.Buffer

	buff.WriteString("((id<GSPromise>(^)(")

	if codegen.notVoid(method.Return) {
		buff.WriteString(fmt.Sprintf("%s", codegen.typeName(method.Return)))
	}

	buff.WriteString("))block)(")

	if codegen.notVoid(method.Return) {
		buff.WriteString("callreturn")
	}

	buff.WriteString(");")

	return buff.String()
}

func enumType(typeDecl ast.Type) string {
	_, ok := gslang.FindAnnotation(typeDecl, "gslang.Flag")

	if !ok {
		return builtin[lexer.KeyUInt32]
	}

	return builtin[lexer.KeyByte]
}

func (codegen *_CodeGen) tagValue(typeDecl ast.Type) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)

		switch builtinType.Type {
		case lexer.KeySByte, lexer.KeyByte, lexer.KeyBool:
			return "GSTagI8"
		case lexer.KeyInt16, lexer.KeyUInt16:
			return "GSTagI16"
		case lexer.KeyInt32, lexer.KeyUInt32, lexer.KeyFloat32:
			return "GSTagI32"
		case lexer.KeyInt64, lexer.KeyUInt64, lexer.KeyFloat64:
			return "GSTagI64"
		case lexer.KeyString:
			return "GSTagString"
		}

	case *ast.TypeRef:
		return codegen.tagValue(typeDecl.(*ast.TypeRef).Ref)
	case *ast.Enum:

		_, ok := gslang.FindAnnotation(typeDecl, "gslang.Flag")

		if !ok {
			return "GSTagI32"
		}

		return "GSTagI8"

	case *ast.Table:
		return "GSTagTable"
	case *ast.Seq:

		seq := typeDecl.(*ast.Seq)

		component := codegen.tagValue(seq.Component)

		if component == "GSTagList" {
			start, _ := gslang.Pos(typeDecl)
			gserrors.Panicf(nil, "list component %v can't be a list :%v", seq.Component, start)
		}

		return fmt.Sprintf("((%s << 4)|GSTagList)", component)
	}

	gserrors.Panicf(nil, "typeName  error: unsupport type(%s)", typeDecl)

	return ""
}

func (codegen *_CodeGen) enumRead(typeDecl ast.Type) string {
	_, ok := gslang.FindAnnotation(typeDecl, "gslang.Flag")

	if ok {
		return fmt.Sprintf("[%s]", readMapping[lexer.KeyUInt32])
	}

	return fmt.Sprintf("[%s]", readMapping[lexer.KeyByte])
}

func (codegen *_CodeGen) enumWrite(typeDecl ast.Type) string {
	_, ok := gslang.FindAnnotation(typeDecl, "gslang.Flag")

	if ok {
		return fmt.Sprintf("[%s:(UInt32) val]", writeMapping[lexer.KeyUInt32])
	}

	return fmt.Sprintf("[%s:(UInt8) val]", writeMapping[lexer.KeyByte])
}

func (codegen *_CodeGen) fieldDecl(field *ast.Field) string {
	return fmt.Sprintf("@property%s %s %s;", propertyAttr(field.Type), codegen.typeName(field.Type), strings.Title(field.Name()))
}

func propertyAttr(typeDecl ast.Type) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:

		builtinType := typeDecl.(*ast.BuiltinType)

		if builtinType.Type == lexer.KeyString {
			return "(nonatomic, strong)"
		}

	case *ast.TypeRef:
		typeRef := typeDecl.(*ast.TypeRef)

		return propertyAttr(typeRef.Ref)

	case *ast.Enum:
		return ""
	case *ast.Table:
		return "(nonatomic, strong)"
	case *ast.Seq:

		return "(nonatomic, strong)"
	}

	return ""
}

func (codegen *_CodeGen) title(typeDecl ast.TypeDecl) string {

	return codegen.typePrefix(typeDecl) + strings.Title(typeDecl.Name())
}

func (codegen *_CodeGen) typePrefix(typeDecl ast.TypeDecl) string {
	langs := gslang.FindAnnotations(typeDecl.Module(), "gslang.Package")

	compiler := codegen.compiler

	for _, lang := range langs {

		langName, ok := lang.Args.NamedArg("Lang")

		if ok && compiler.Eval().EvalString(langName) == "objc" {

			packageName, ok := lang.Args.NamedArg("Name")

			if ok && compiler.Eval().EvalString(packageName) == typeDecl.Package() {

				redirect, ok := lang.Args.NamedArg("Redirect")

				if ok {
					prefix := compiler.Eval().EvalString(redirect)

					if codegen.script.Name() != typeDecl.Script() {
						v := fmt.Sprintf("#import <%s>\n\n", filepath.Join(strings.Replace(typeDecl.Package(), ".", "/", -1), filepath.Base(typeDecl.Script())+".h"))

						codegen.imports[typeDecl.FullName()] = v
					}

					return prefix
				}
			}

		}
	}

	return ""
}

func (codegen *_CodeGen) enumFields(enum *ast.Enum) string {
	var buff bytes.Buffer

	for _, v := range enum.Constants {
		buff.WriteString(fmt.Sprintf("\n\t%s = %d,", codegen.title(enum)+strings.Title(v.Name()), v.Value))
	}

	content := buff.String()

	return content[:len(content)-1] + "\n"
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
		return codegen.title(typeDecl.(ast.TypeDecl))
	case *ast.Table:

		return codegen.title(typeDecl.(ast.TypeDecl)) + "*"

	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		isbytes := false

		builtinType, ok := seq.Component.(*ast.BuiltinType)

		if ok && builtinType.Type == lexer.KeyByte {
			isbytes = true
		}

		if isbytes {
			return "NSMutableData *"
		}

		return "NSMutableArray *"
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

		return codegen.defaultVal(typeDecl.(*ast.TypeRef).Ref)

	case *ast.Enum:
		enum := typeDecl.(*ast.Enum)

		return codegen.title(enum) + strings.Title(enum.Constants[0].Name())

	case *ast.Table:

		return fmt.Sprintf("[[%s alloc] init]", codegen.title(typeDecl.(ast.TypeDecl)))

	case *ast.Seq:

		seq := typeDecl.(*ast.Seq)

		isbytes := false

		builtinType, ok := seq.Component.(*ast.BuiltinType)

		if ok && builtinType.Type == lexer.KeyByte {
			isbytes = true
		}

		if isbytes {
			return "[[NSMutableData alloc] init]"
		}

		return "[NSMutableArray arrayWithCapacity: 0]"
	}

	return "UNKNOWN"
}

func writeindent(stream *bytes.Buffer, indent int) {
	for i := 0; i < indent; i++ {
		stream.WriteRune('\t')
	}
}

func (codegen *_CodeGen) marshal(varname string, typeDecl ast.Type, indent int) string {
	var stream bytes.Buffer

	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		writeindent(&stream, indent)
		stream.WriteString(fmt.Sprintf("[%s :%s];", writeMapping[builtinType.Type], varname))
	case *ast.TypeRef:
		typeRef := typeDecl.(*ast.TypeRef)

		return codegen.marshal(varname, typeRef.Ref, indent)
	case *ast.Table:
		writeindent(&stream, indent)
		stream.WriteString(fmt.Sprintf(
			"[%s marshal: writer];",
			varname,
		))
	case *ast.Enum:
		writeindent(&stream, indent)
		stream.WriteString(fmt.Sprintf(
			"[%sHelper marshal: %s withWriter: writer];",
			codegen.typeName(typeDecl),
			varname,
		))
	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		isbytes := false

		builtinType, ok := seq.Component.(*ast.BuiltinType)

		if ok && builtinType.Type == lexer.KeyByte {
			isbytes = true
		}

		writeindent(&stream, indent)

		if isbytes {
			stream.WriteString(fmt.Sprintf("[writer WriteBytes: %s];", varname))
			break
		}

		if seq.Size == -1 {

			stream.WriteString(fmt.Sprintf("[writer WriteUInt16:%s.count];\n", varname))

			writeindent(&stream, indent)

			stream.WriteString(fmt.Sprintf("for(id v%d in %s){\n", indent, varname))

			writeindent(&stream, indent+1)

			stream.WriteString(fmt.Sprintf("%s vv%d = %s;\n", codegen.typeName(seq.Component), indent, codegen.toComponentType(fmt.Sprintf("v%d", indent), seq.Component)))

			stream.WriteString(codegen.marshal(fmt.Sprintf("vv%d", indent), seq.Component, indent+1))

			writeindent(&stream, indent)

			stream.WriteRune('}')

		} else {

			stream.WriteString(fmt.Sprintf("for(id v%d in %s){\n", indent, varname))

			writeindent(&stream, indent+1)

			stream.WriteString(fmt.Sprintf("%s vv%d = %s;\n", codegen.typeName(seq.Component), indent, codegen.toComponentType(fmt.Sprintf("v%d", indent), seq.Component)))

			stream.WriteString(codegen.marshal(fmt.Sprintf("vv%d", indent), seq.Component, indent+1))

			writeindent(&stream, indent)

			stream.WriteRune('}')
		}

	}

	stream.WriteRune('\n')

	return stream.String()
}

func (codegen *_CodeGen) toComponentType(varname string, typeDecl ast.Type) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		return fmt.Sprintf(id2Type[builtinType.Type], varname)
	case *ast.TypeRef, *ast.Table, *ast.Enum, *ast.Seq:
		return fmt.Sprintf("(%s)%s", codegen.typeName(typeDecl), varname)
	}

	return ""
}

func (codegen *_CodeGen) fromComponentType(varname string, typeDecl ast.Type) string {
	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		return fmt.Sprintf(type2id[builtinType.Type], varname)
	case *ast.TypeRef, *ast.Table, *ast.Enum, *ast.Seq:
		return varname
	}

	return ""
}

func (codegen *_CodeGen) unmarshal(varname string, typeDecl ast.Type, indent int) string {
	var stream bytes.Buffer

	switch typeDecl.(type) {
	case *ast.BuiltinType:
		builtinType := typeDecl.(*ast.BuiltinType)
		writeindent(&stream, indent)
		stream.WriteString(fmt.Sprintf("%s = [%s];", varname, readMapping[builtinType.Type]))
	case *ast.TypeRef:
		typeRef := typeDecl.(*ast.TypeRef)

		return codegen.unmarshal(varname, typeRef.Ref, indent)
	case *ast.Table:
		writeindent(&stream, indent)
		stream.WriteString(fmt.Sprintf(
			"[%s unmarshal:reader ];",
			varname,
		))
	case *ast.Enum:
		writeindent(&stream, indent)
		stream.WriteString(fmt.Sprintf(
			"%s = [%sHelper unmarshal: reader];",
			varname,
			codegen.typeName(typeDecl),
		))
	case *ast.Seq:
		seq := typeDecl.(*ast.Seq)

		isbytes := false

		builtinType, ok := seq.Component.(*ast.BuiltinType)

		if ok && builtinType.Type == lexer.KeyByte {
			isbytes = true
		}

		writeindent(&stream, indent)

		if seq.Size == -1 {

			if isbytes {
				stream.WriteString(fmt.Sprintf("%s = [reader ReadBytes];", varname))
				break
			}

			stream.WriteString(fmt.Sprintf("UInt16 imax%d = [reader ReadUInt16];\n\n", indent))

			writeindent(&stream, indent)

			stream.WriteString(fmt.Sprintf("for(UInt16 i%d = 0; i%d < imax%d; i%d ++ ){\n\n", indent, indent, indent, indent))

			writeindent(&stream, indent+1)

			stream.WriteString(fmt.Sprintf("%s v%d = %s;\n\n", codegen.typeName(seq.Component), indent, codegen.defaultVal(seq.Component)))

			stream.WriteString(codegen.unmarshal(fmt.Sprintf("v%d", indent), seq.Component, indent+1))

			stream.WriteRune('\n')

			writeindent(&stream, indent+1)

			stream.WriteString(fmt.Sprintf("[ %s addObject: %s];\n\n", varname, codegen.fromComponentType(fmt.Sprintf("v%d", indent), seq.Component)))

			writeindent(&stream, indent)

			stream.WriteRune('}')
		} else {

			if isbytes {
				stream.WriteString(fmt.Sprintf("%s = [reader ReadArrayBytes:%d];", varname, seq.Size))
				break
			}

			stream.WriteString(fmt.Sprintf("UInt16 imax%d = [reader ReadUInt16];\n\n", indent))

			writeindent(&stream, indent)

			stream.WriteString(fmt.Sprintf("for(UInt16 i%d = 0; i%d < imax%d; i%d ++ ){\n\n", indent, indent, indent, indent))

			writeindent(&stream, indent+1)

			stream.WriteString(fmt.Sprintf("%s v%d = %s;\n\n", codegen.typeName(seq.Component), indent, codegen.defaultVal(seq.Component)))

			stream.WriteString(codegen.unmarshal(fmt.Sprintf("v%d", indent), seq.Component, indent+1))

			stream.WriteRune('\n')

			writeindent(&stream, indent+1)

			stream.WriteString(fmt.Sprintf("[ %s addObject: %s];\n\n", varname, codegen.fromComponentType(fmt.Sprintf("v%d", indent), seq.Component)))

			writeindent(&stream, indent)

			stream.WriteRune('}')
		}

	}

	stream.WriteRune('\n')

	return stream.String()
}

func (codegen *_CodeGen) marshalField(field *ast.Field) string {
	return codegen.marshal("_"+strings.Title(field.Name()), field.Type, 1)
}

func (codegen *_CodeGen) unmarshalField(field *ast.Field) string {
	return codegen.unmarshal("_"+strings.Title(field.Name()), field.Type, 1)
}

func (codegen *_CodeGen) methodDecl(method *ast.Method) string {

	var stream bytes.Buffer

	stream.WriteString(fmt.Sprintf("- (%s)", codegen.typeName(method.Return)))

	if len(method.Params) > 0 {
		stream.WriteString(fmt.Sprintf(" %s:(%s)arg0", method.Name(), codegen.typeName(method.Params[0].Type)))

		for i := 1; i < len(method.Params); i++ {
			stream.WriteString(fmt.Sprintf(" withArg%d:(%s)arg%d", i, codegen.typeName(method.Params[i].Type), i))
		}

	} else {
		stream.WriteString(fmt.Sprintf(" %s", method.Name()))
	}

	return stream.String()
}

func (codegen *_CodeGen) rpcMethodDecl(method *ast.Method) string {

	var stream bytes.Buffer
	if gslang.IsAsync(method) {
		stream.WriteString("- (NSError*)")
	} else {
		stream.WriteString("- (id<GSPromise>)")
	}

	if len(method.Params) > 0 {
		stream.WriteString(fmt.Sprintf(" %s:(%s) arg0 ", method.Name(), codegen.typeName(method.Params[0].Type)))

		for i := 1; i < len(method.Params); i++ {
			stream.WriteString(fmt.Sprintf(" withArg%d:(%s) arg%d ", i, codegen.typeName(method.Params[i].Type), i))
		}

	} else {
		stream.WriteString(fmt.Sprintf(" %s", method.Name()))
	}

	return stream.String()
}

func (codegen *_CodeGen) marshalParam(param *ast.Param, varname string, indent int) string {
	var buff bytes.Buffer

	writeindent(&buff, indent)

	buff.WriteString("{\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString("GSBytesWriter *writer = [[GSBytesWriter alloc] init];\n\n")

	buff.WriteString(codegen.marshal(varname, param.Type, indent+1))

	writeindent(&buff, indent+1)

	buff.WriteString("GSParam *param  = [GSParam init];\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString("param.Content = writer.content;\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString("[params addObject:param];\n\n")

	writeindent(&buff, indent)

	buff.WriteString("}\n\n")
	return buff.String()
}

func (codegen *_CodeGen) marshalReturn(typeDecl ast.Type) string {

	var buff bytes.Buffer

	indent := 2

	writeindent(&buff, indent)

	buff.WriteString("{\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString("GSBytesWriter *writer = [[GSBytesWriter alloc] init];\n\n")

	buff.WriteString(codegen.marshal("ret", typeDecl, indent+1))

	writeindent(&buff, indent+1)

	buff.WriteString("callreturn.Content = writer.content;\n\n")

	writeindent(&buff, indent)

	buff.WriteString("}\n\n")
	return buff.String()
}

func (codegen *_CodeGen) unmarshalReturn(typeDecl ast.Type, indent int) string {
	var buff bytes.Buffer

	writeindent(&buff, indent)

	buff.WriteString(fmt.Sprintf("%s callreturn = %s;\n\n", codegen.typeName(typeDecl), codegen.defaultVal(typeDecl)))

	writeindent(&buff, indent)

	buff.WriteString("{\n\n")

	writeindent(&buff, indent+1)

	buff.WriteString("GSBytesReader *reader = [GSBytesReader initWithNSData: response.Content];\n\n")

	buff.WriteString(codegen.unmarshal("callreturn", typeDecl, indent+1))

	writeindent(&buff, indent)

	buff.WriteString("}\n\n")

	return buff.String()
}

func (codegen *_CodeGen) unmarshalParam(param *ast.Param, varname string) string {
	var buff bytes.Buffer

	writeindent(&buff, 3)

	buff.WriteString(fmt.Sprintf("%s arg%d = %s;\n\n", codegen.typeName(param.Type), param.ID, codegen.defaultVal(param.Type)))

	writeindent(&buff, 3)

	buff.WriteString("{\n\n")

	writeindent(&buff, 4)

	buff.WriteString(fmt.Sprintf("GSBytesReader *reader = [GSBytesReader initWithNSData: ((GSParam*)%s.Params[%d]).Content];\n\n", varname, param.ID))

	buff.WriteString(codegen.unmarshal(fmt.Sprintf("arg%d", param.ID), param.Type, 4))

	writeindent(&buff, 3)

	buff.WriteString("}\n\n")

	return buff.String()
}

func (codegen *_CodeGen) notVoid(typeDecl ast.Type) bool {
	builtinType, ok := typeDecl.(*ast.BuiltinType)

	if !ok {
		return true
	}

	return builtinType.Type != lexer.KeyVoid
}

func (codegen *_CodeGen) methodCall(method *ast.Method) string {

	var stream bytes.Buffer

	if codegen.notVoid(method.Return) {
		stream.WriteString(fmt.Sprintf("%s ret = ", codegen.typeName(method.Return)))
	}

	if len(method.Params) > 0 {
		stream.WriteString(fmt.Sprintf("[ _service %s: arg0 ", method.Name()))

		for i := 1; i < len(method.Params); i++ {
			stream.WriteString(fmt.Sprintf(" withArg%d:arg%d ", i, i))
		}

	} else {
		stream.WriteString(fmt.Sprintf(" [ _service %s", method.Name()))
	}

	stream.WriteString("];")

	return stream.String()
}

func (codegen *_CodeGen) marshalParams(params []*ast.Param) string {

	var buff bytes.Buffer

	for _, param := range params {

		buff.WriteString(codegen.marshalParam(param, fmt.Sprintf("arg%d", param.ID), 1))
	}

	return buff.String()
}

func (codegen *_CodeGen) BeginScript(compiler *gslang.Compiler, script *ast.Script) bool {

	codegen.compiler = compiler

	scriptPath := filepath.ToSlash(filepath.Clean(script.Name()))

	for _, skip := range codegen.skips {

		if skip.MatchString(scriptPath) {
			return false
		}
	}

	if strings.HasPrefix(script.Package, "gslang") {
		return false
	}

	codegen.header.Reset()
	codegen.source.Reset()
	codegen.predecl.Reset()

	codegen.script = script

	codegen.imports = make(map[string]string)

	for k, v := range imports {
		codegen.imports[k] = v
	}

	return true
}

func (codegen *_CodeGen) Using(compiler *gslang.Compiler, using *ast.Using) {

}

func (codegen *_CodeGen) Table(compiler *gslang.Compiler, tableType *ast.Table) {

	if err := codegen.tpl.ExecuteTemplate(&codegen.predecl, "table_predecl", tableType); err != nil {
		gserrors.Panicf(err, "exec template(table_predecl) for %s error", tableType)
	}

	if err := codegen.tpl.ExecuteTemplate(&codegen.header, "table_header", tableType); err != nil {

		gserrors.Panicf(err, "exec template(table) for %s error", tableType)
	}

	if err := codegen.tpl.ExecuteTemplate(&codegen.source, "table_source", tableType); err != nil {
		gserrors.Panicf(err, "exec template(table) for %s error", tableType)
	}
}

func (codegen *_CodeGen) Annotation(compiler *gslang.Compiler, annotation *ast.Table) {
}

func (codegen *_CodeGen) Enum(compiler *gslang.Compiler, enum *ast.Enum) {

	if err := codegen.tpl.ExecuteTemplate(&codegen.predecl, "enum_predecl", enum); err != nil {
		gserrors.Panicf(err, "exec template(enum_predecl) for %s error", enum)
	}

	if err := codegen.tpl.ExecuteTemplate(&codegen.header, "enum_header", enum); err != nil {
		gserrors.Panicf(err, "exec template(Enum) for %s error", enum)
	}

	if err := codegen.tpl.ExecuteTemplate(&codegen.source, "enum_source", enum); err != nil {
		gserrors.Panicf(err, "exec template(Enum) for %s error", enum)
	}
}
func (codegen *_CodeGen) Contract(compiler *gslang.Compiler, contract *ast.Contract) {
	if err := codegen.tpl.ExecuteTemplate(&codegen.header, "contract_header", contract); err != nil {
		gserrors.Panicf(err, "exec template(Contract) for %s error", contract)
	}

	if err := codegen.tpl.ExecuteTemplate(&codegen.source, "contract_source", contract); err != nil {
		gserrors.Panicf(err, "exec template(Contract) for %s error", contract)
	}
}

func (codegen *_CodeGen) writefile(bytes []byte, extend string) {

	path := strings.Replace(codegen.script.Package, ".", "/", -1)

	fullpath := filepath.Join(codegen.rootpath, path, filepath.Base(codegen.script.Name())+extend)

	if err := os.MkdirAll(filepath.Dir(fullpath), 0755); err != nil {
		gserrors.Panicf(err, "create output directory error")
	}

	if err := ioutil.WriteFile(fullpath, bytes, 0644); err != nil {
		gserrors.Panicf(err, "write generate stub code error")
	}
}

// EndScript .
func (codegen *_CodeGen) EndScript(compiler *gslang.Compiler) {

	var stream bytes.Buffer

	guard := strings.ToUpper(strings.Replace(path.Join(codegen.script.Package, filepath.Base(codegen.script.Name())), ".", "_", -1))

	guard = strings.Replace(guard, "/", "_", -1)

	stream.WriteString(fmt.Sprintf("#ifndef %s\n", guard))
	stream.WriteString(fmt.Sprintf("#define %s\n", guard))

	imports := make(map[string]string)

	for _, i := range codegen.imports {
		imports[i] = i
	}

	for k := range imports {
		stream.WriteString(fmt.Sprintf("%s\n\n", k))
	}

	stream.Write(codegen.predecl.Bytes())

	stream.Write(codegen.header.Bytes())

	stream.WriteString(fmt.Sprintf("\n#endif //%s\n", guard))

	codegen.writefile(stream.Bytes(), ".h")

	stream.Reset()

	path := filepath.Join(strings.Replace(codegen.script.Package, ".", "/", -1), filepath.Base(codegen.script.Name())+".h")

	stream.WriteString(fmt.Sprintf("#import <%s>\n\n", path))

	stream.WriteString("#import <com/gsrpc/gsrpc.gs.h>\n\n")

	stream.Write(codegen.source.Bytes())

	codegen.writefile(stream.Bytes(), ".m")
}
