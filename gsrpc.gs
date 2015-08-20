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
