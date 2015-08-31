package gen4java

var t4java = `
{{define "enum"}}{{$Enum := title .Name}}
/*
 * {{title .Name}} generate by gs2java,don't modify it manually
 */
public enum {{title .Name}} {
    {{enumFields .}}
    private {{enumType .}} value;
    {{title .Name}}({{enumType .}} val){
        this.value = val;
    }
    @Override
    public String toString() {
        switch(this.value)
        {
        {{range .Values}}
        case {{.Value}}:
            return "{{title .Name}}";
        {{end}}
        }
        return String.format("{{title .Name}}#%d",this.value);
    }
    public {{enumType .}} getValue() {
        return this.value;
    }
    public void Marshal(Writer writer) throws Exception
    {
        return {{if enumSize . | eq 4}} writer.WriteUint32(writer,val.getValue()); {{else}} writer.WriteByte(writer,val.getValue()); {{end}}
    }
    public static {{title .Name}} Unmarshal(Reader reader) throws Exception
    {
        {{enumType .}} code =  {{if enumSize . | eq 4}} reader.ReadUint32(reader) {{else}} reader.ReadByte(reader) {{end}}
        switch(code)
        {
        {{range .Values}}
        case {{.Value}}:
            return {{$Enum}}.{{title .Name}};
        {{end}}
        }
        throw new Exception("unknown enum constant :" + code);
    }
}
{{end}}

{{define "table"}}{{$Struct := title .Name}}
/*
 * {{title .Name}} generate by gs2java,don't modify it manually
 */
public class {{$Struct}}
{
{{range .Fields}}
    private  {{typeName .Type}} {{fieldName .Name}} = {{defaultVal .Type}};
{{end}}

{{range .Fields}}
    public {{typeName .Type}} get{{title .Name}}()
    {
        return this.{{fieldName .Name}};
    }
    public void set{{title .Name}}({{typeName .Type}} arg)
    {
        this.{{fieldName .Name}} = arg;
    }
{{end}}
    public void Marshal(Writer writer)  throws Exception
    {
{{range .Fields}}
{{marshalField .}}
{{end}}
    }
    public void Unmarshal(Reader reader) throws Exception
    {
{{range .Fields}}
{{unmarshalField .}}
{{end}}
    }
}
{{end}}


{{define "Service"}}{{$Contract := title .Name}}

public interface {{$Contract}} {
{{range .Methods}}
    {{resultype .Return}} {{title .Name}}({{paramsDecl .Params}}) throws Exception;
{{end}}
}
{{end}}
{{define "AbstractService"}}
{{$Contract := title .Name}}
/*
 * {{title .Name}} generate by gs2java,don't modify it manually
 */
public final class {{$Contract}}Dispatcher implements com.github.gsdocker.gsrpc.Dispatcher {
    private {{$Contract}} service;
    public {{$Contract}}Dispatcher({{$Contract}} service) {
        this.service = service;
    }
    public com.github.gsdocker.gsrpc.Return Dispatch(com.github.gsdocker.gsrpc.Call call) throws Exception
    {
        switch(call.getMethod()){
        {{range .Methods}}
        case {{.ID}}: {
{{range .Params}}{{unmarshalParam . "call" 4}}{{end}}
                {{methodcall .}}
                {{if .Return}}
                com.github.gsdocker.gsrpc.Param[] params = new com.github.gsdocker.gsrpc.Param[{{params .Return}}];
{{marshalReturnParams .Return}}
                com.github.gsdocker.gsrpc.Return callReturn = new com.github.gsdocker.gsrpc.Return();
                callReturn.setID(call.getID());
                callReturn.setService(call.getService());
                callReturn.setParams(params);
                return callReturn;
                {{else}}
                break;
                {{end}}
            }
        {{end}}
        }
        return null;
    }
}
{{end}}
{{define "RPC"}}
{{$Contract := title .Name}}
/*
 * {{title .Name}} generate by gs2java,don't modify it manually
 */
public final class {{$Contract}}RPC {
    private com.github.gsdocker.gsrpc.Net net;
    private short serviceid;
    public {{$Contract}}RPC(com.github.gsdocker.gsrpc.Net net, short serviceid){
        this.net = net;
        this.serviceid = serviceid;
    }
    {{range .Methods}}
    public {{methodRPC .}} throws Exception {
        com.github.gsdocker.gsrpc.Call call = new com.github.gsdocker.gsrpc.Call();
        call.setService(this.serviceid);
        call.setMethod((short){{.ID}});
        {{if .Params}}
        com.github.gsdocker.gsrpc.Param[] params = new com.github.gsdocker.gsrpc.Param[{{params .Params}}];
{{marshalParams .Params}}
        call.setParams(params);
        {{end}}
        this.net.send(call,new com.github.gsdocker.gsrpc.Callback(){
            @Override
            public int getTimeout() {
                return timeout;
            }
            @Override
            public void Return(Exception e,com.github.gsdocker.gsrpc.Return callReturn){
                if (e != null) {
                    completeHandler.Complete(e);
                    return;
                }
                try{
{{range .Return}}{{unmarshalParam . "callReturn" 5}}{{end}}
                    {{callback .}}
                }catch(Exception e1) {
                    completeHandler.Complete(e1);
                }
            }
        });
    }
    {{end}}
}
{{end}}
`
