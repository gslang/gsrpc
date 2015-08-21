package com.gsrpc.test;

using gslang.annotations.Usage;
using gslang.annotations.Target;
using gslang.Exception;
using gslang.Flag;
using com.gsrpc.Message;


enum TimeUnit{
    Second
}

table Duration {
    int32 Value;
    TimeUnit Unit;
}


// Description define new Attribute
@Usage(Target.Package|Target.Script)
table Description {
    string Text; // Description text
    //long texts
    string LongText;
}

@Usage(Target.Method)
table Async {
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
    Message message;
}

table KV {
    string Key;
    string Value;
}



// HTTPREST API
contract HTTPREST {
    @Async
    // invoke http post method
    @Timeout(Duration(-100,TimeUnit.Second))
    void Post(@Out byte[] content) throws (RemoteException,CodeException);
    // get invoke http get method
    byte[] Get(KV[] properties) throws (RemoteException);
    void PostMessage(Message message);
}

table Block {
    byte[256] Content;
    KV[12][128] KV;
}

// remote exception
@Flag
@Exception
table CodeException {
}
