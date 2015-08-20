package gen4go

var tpl = `

{{define "enum"}} {{$Enum := title .Name}}

//{{$Enum}} type define -- generate by gsc
type {{$Enum}} {{enumType .}}

//enum {{$Enum}} constants -- generate by gsc
const (
    {{range .Constants}}
    {{$Enum}}{{title .Name}} {{$Enum}} = {{.Value}}
    {{end}}
)

//Write{{$Enum}} write enum to output stream
func Write{{$Enum}}(writer gorpc.Writer, val {{$Enum}}) error{
    return {{if enumSize . | eq 4}} gorpc.WriteUint32(writer,uint32(val)) {{else}} gorpc.WriteByte(writer,byte(val)) {{end}}
}

//Read{{$Enum}} write enum to output stream
func Read{{$Enum}}(reader gorpc.Reader)({{$Enum}}, error){
    val,err := {{if enumSize . | eq 4}} gorpc.ReadUint32(reader) {{else}} gorpc.ReadByte(reader) {{end}}
    return {{$Enum}}(val),err
}

//String implement Stringer interface
func (val {{$Enum}}) String() string {
    switch val {
        {{range .Constants}}
        case {{.Value}}:
            return "enum({{$Enum}}.{{title .Name}})"
        {{end}}
    }
    return fmt.Sprintf("enum(Unknown(%d))",val)
}

{{end}}

{{define "exception"}} {{$Table := title .Name}}

//{{$Table}} -- generate by gsc
type {{$Table}} struct {
    {{range .Fields}}
    {{title .Name}} {{typeName .Type}}
    {{end}}
}

//New{{$Table}} create new struct object with default field val -- generate by gsc
func New{{$Table}}() *{{$Table}} {
    return &{{$Table}}{
        {{range .Fields}}
        {{title .Name}}: {{defaultVal .Type}},
        {{end}}
    }
}

{{end}}

{{define "table"}} {{$Table := title .Name}}

//{{$Table}} -- generate by gsc
type {{$Table}} struct {
    {{range .Fields}}
    {{title .Name}} {{typeName .Type}}
    {{end}}
}

//New{{$Table}} create new struct object with default field val -- generate by gsc
func New{{$Table}}() *{{$Table}} {
    return &{{$Table}}{
        {{range .Fields}}
        {{title .Name}}: {{defaultVal .Type}},
        {{end}}
    }
}

//Read{{$Table}} read {{$Table}} from input stream -- generate by gsc
func Read{{$Table}}(reader gorpc.Reader) (target *{{$Table}},err error) {
    target = New{{$Table}}()
    {{range .Fields}}
    target.{{title .Name}},err = {{if isBuiltin .Type}}{{else}}{{end}}
    if err != nil {
        return
    }
    {{end}}
    return
}


{{end}}

{{define "annotation"}}

{{end}}


{{define "contract"}}

{{end}}


{{define "create_list"}}nil{{end}}

{{define "create_array"}}func() {{typeName .}} {

    var buff {{typeName .}}

    {{if isBuiltin .Component}}
    {{else}}
    for i := uint16(0); i < {{.Size}}; i ++ {
        buff[i] = {{defaultVal .Component}}
    }
    {{end}}

    return buff
}(){{end}}

`
