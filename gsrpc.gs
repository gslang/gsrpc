package com.gsrpc;

// RPC message codes
enum Code {
    Heartbeat,WhoAmI,Request,Response,Accept,Reject
}

enum State{
    Disconnect,Connecting,Connected,Disconnecting,Closed
}

// RPC message
table Message {
    Code    Code;       // message code
    byte    Agent;      // message agent id
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
    sbyte       Exception; // exception id
    byte[]      Content;
}

enum OSType {
    Windows(0),Linux(1),OSX(2),WP(3),Android(4),iOS(5)
}

enum ArchType {
    X86(0),X64(1),ARM(2)
}

// The client device type
table Device {
    string      ID;             // device udid
    string      Type;           // device type
    ArchType    Arch;           // device arch type
    OSType      OS;             // device os type
    string      OSVersion;      // device os reversion
}

table WhoAmI {
    Device      ID;             // device name
    byte[]      Context;         // context data
}
