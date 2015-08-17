package gen4go

var tpl = `

{{define "enum"}} {{$Enum := .Name}}

//{{$Enum}} type define -- generate by gsc
type {{$Enum}} {{enumType .}}

//enum {{$Enum}} constants -- generate by gsc
const (
    {{range .Constants}}
    {{$Enum}}{{.Name}} {{$Enum}} = {{.Value}}
    {{end}}
)

{{end}}

{{define "table"}}

{{end}}

{{define "annotation"}}

{{end}}


{{define "contract"}}

{{end}}


`
