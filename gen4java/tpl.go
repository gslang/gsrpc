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
        {{range .Constants}}
        case {{.Value}}:
            return "{{title .Name}}";
        {{end}}
        }
        return "{{title .Name}}#" + this.value;
    }
    public {{enumType .}} getValue() {
        return this.value;
    }
    public void marshal(Writer writer) throws Exception
    {
        {{if enumSize . | eq 4}} writer.writeUInt32(getValue()); {{else}} writer.writeByte(getValue()); {{end}}
    }
    public static {{title .Name}} unmarshal(Reader reader) throws Exception
    {
        {{enumType .}} code =  {{if enumSize . | eq 4}} reader.readUInt32(); {{else}} reader.readByte(); {{end}}
        switch(code)
        {
        {{range .Constants}}
        case {{.Value}}:
            return {{$Enum}}.{{title .Name}};
        {{end}}
        }
        throw new Exception("unknown enum constant :" + code);
    }
}
{{end}}

{{define "table"}}{{$Struct := tableName .}}
public class {{$Struct}} {{if isException .}}extends Exception{{end}}
{
{{range .Fields}}
    private  {{typeName .Type}} {{fieldName .Name}} = {{defaultVal .Type}};
{{end}}

{{if .Fields}}
    public {{$Struct}}(){

    }
{{end}}

    public {{$Struct}}{{constructor .Fields}} {
    {{range .Fields}}
        this.{{fieldName .Name}} = {{fieldName .Name}};
    {{end}}
    }

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

{{if isPOD . | not}}
    public void marshal(Writer writer)  throws Exception
    {
        writer.writeByte((byte){{len .Fields}});
{{range .Fields}}
        writer.writeByte((byte){{tagValue .Type}});
        {{marshalField .}}
{{end}}
    }
    public void unmarshal(Reader reader) throws Exception
    {
        byte __fields = reader.readByte();
{{range .Fields}}
        {
            byte tag = reader.readByte();

            if(tag != com.gsrpc.Tag.Skip.getValue()) {
                {{unmarshalField .}}
            }

            if(-- __fields == 0) {
                return;
            }
        }

{{end}}

        for(int i = 0; i < (int)__fields; i ++) {
            byte tag = reader.readByte();

            if (tag == com.gsrpc.Tag.Skip.getValue()) {
                continue;
            }

            reader.readSkip(tag);
        }
    }
{{else}}
    public void marshal(Writer writer)  throws Exception
    {
{{range .Fields}}
        {{marshalField .}}
{{end}}
    }

    public void unmarshal(Reader reader) throws Exception
    {
{{range .Fields}}
        {
            {{unmarshalField .}}
        }
{{end}}
    }

{{end}}
}
{{end}}


{{define "contract"}}{{$Contract := title .Name}}

public interface {{$Contract}} {
    String NAME = "{{.FullName}}";
{{range .Methods}}
    {{returnParam .Return}} {{methodName .Name}} {{params .Params}} throws Exception;
{{end}}
}

{{end}}


{{define "dispatcher"}}
{{$Contract := title .Name}}
/*
 * {{title .Name}} generate by gs2java,don't modify it manually
 */
public final class {{$Contract}}Dispatcher implements com.gsrpc.NamedDispatcher {

    private {{$Contract}} service;

    public {{$Contract}}Dispatcher({{$Contract}} service) {
        this.service = service;
    }

    public String name() {
        return "{{.FullName}}";
    }

    public com.gsrpc.Response dispatch(com.gsrpc.Request call) throws Exception
    {
        switch(call.getMethod()){
        {{range .Methods}}
        case {{.ID}}: {
{{range .Params}}{{unmarshalParam . "call" 4}}{{end}}
                {{if isAsync . | not}}
                {{if .Exceptions}}
                try{
                {{end}}
                    {{methodcall .}}

                    com.gsrpc.Response callReturn = new com.gsrpc.Response();
                    callReturn.setID(call.getID());
                    callReturn.setException((byte)-1);

                    {{if notVoid .Return}}
    {{marshalReturn .Return "ret" 4}}
                    callReturn.setContent(returnParam);
                    {{end}}

                    return callReturn;

                {{if .Exceptions}}}{{end}}{{range .Exceptions}} catch({{typeName .Type}} e) {

                    com.gsrpc.BufferWriter writer = new com.gsrpc.BufferWriter();

                    e.marshal(writer);

                    com.gsrpc.Response callReturn = new com.gsrpc.Response();
                    callReturn.setID(call.getID());
                    callReturn.setException((byte){{.ID}});
                    callReturn.setContent(writer.getContent());

                    return callReturn;
                }{{end}}
                {{else}}
                {{methodcall .}}
                {{end}}
            }
        {{end}}
        }
        return null;
    }
}
{{end}}


{{define "rpc"}}{{$Contract := title .Name}}
/*
 * {{title .Name}} generate by gs2java,don't modify it manually
 */
public final class {{$Contract}}RPC {

    /**
     * gsrpc net interface
     */
    private com.gsrpc.Channel net;

    /**
     * remote service id
     */
    private short serviceID;

    public {{$Contract}}RPC(com.gsrpc.Channel net, short serviceID){
        this.net = net;
        this.serviceID = serviceID;
    }

    {{range .Methods}}{{$Name := title .Name}}
    public {{methodRPC .}} throws Exception {

        com.gsrpc.Request request = new com.gsrpc.Request();

        request.setService(this.serviceID);

        request.setMethod((short){{.ID}});

        {{if .Params}}
        com.gsrpc.Param[] params = new com.gsrpc.Param[{{len .Params}}];
{{marshalParams .Params}}
        request.setParams(params);
        {{end}}

        {{if isAsync . | not}}
        com.gsrpc.Promise<{{objTypeName .Return}}> promise = new com.gsrpc.Promise<{{objTypeName .Return}}>(timeout){
            @Override
            public void Return(Exception e,com.gsrpc.Response callReturn){

                if (e != null) {
                    Notify(e,null);
                    return;
                }

                try{

                    if(callReturn.getException() != (byte)-1) {
                        switch(callReturn.getException()) {
                            {{range .Exceptions}}
                            case {{.ID}}:{
                            com.gsrpc.BufferReader reader = new com.gsrpc.BufferReader(callReturn.getContent());

                            {{typeName .Type}} exception = {{defaultVal .Type}};

                            {{readType "exception" .Type 4}}

                            Notify(exception,null);

                            return;
                        }
                        {{end}}
                        default:
                            Notify(new com.gsrpc.RemoteException(),null);
                            return;
                        }
                    }

                    {{if notVoid .Return}}
{{unmarshalReturn .Return "callReturn" 5}}
                    Notify(null,returnParam);
                    {{else}}
                    Notify(null,null);
                    {{end}}
                }catch(Exception e1) {
                    Notify(e1,null);
                }
            }
        };

        this.net.send(request,promise);

        return promise;
        {{else}}
        this.net.post(request);
        {{end}}
    }
    {{end}}
}
{{end}}
`
