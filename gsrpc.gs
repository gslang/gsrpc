package com.gsrpc;

// RPC message codes
enum Code {
    Heartbeat,WhoAmI,Call,Return,Exception
}

// RPC message
table Message {
    Code    Code;       // message code
    byte[]  Content;    // message content
}

table Param {
    byte[] Content;
}

table Request {
    uint16      ID;
    uint16      Method;
    uint16      Service;
    Param[]     Params;
}

table Response {
    uint16      ID;
    uint16      Service;
    uint16      Exception; // exception id
    byte[]      Content;
}
