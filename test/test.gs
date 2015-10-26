package com.gsrpc.test;

using gslang.annotations.Usage;
using gslang.annotations.Target;
using gslang.Exception;
using gslang.Flag;
using gslang.Package;

@Package(Lang:"objc",Name:"com.gsrpc.test",Redirect:"GSTest")
@Package(Lang:"golang",Name:"com.gsrpc.test",Redirect:"github.com/gsrpc/gorpc/test")

enum TimeUnit{
    Second
}

table Duration {
    int32 Value;
    TimeUnit Unit;
}


// Description define new Attribute
@Usage(Target.Module|Target.Script)
table Description {
    string Text; // Description text
    //long texts
    string LongText;
}

@Usage(Target.Param)
table Out {
}

@Usage(Target.Method)
table Timeout {
    Duration Duration;
}

// remote exception
@Exception
table RemoteException {

}

table KV {
    string Key;
    string Value;
}



// RESTful API
contract RESTful {

    // invoke http post method
    @Timeout(Duration(-100,TimeUnit.Second))
    void Post(string name,byte[] content) throws (RemoteException,NotFound);
    // get invoke http get method
    byte[] Get(string name) throws (NotFound);

    @gslang.Async
    void SayHello(string message);
}

table Block {
    byte[256] Content;
    KV[12][128] KV;
}

// remote exception
@Flag
@Exception
table NotFound {
}
