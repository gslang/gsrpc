package com.gsrpc;

using gslang.Package;
using gslang.Exception;
using gslang.Flag;

@Package(Lang:"golang",Name:"com.gsrpc",Redirect:"github.com/gsrpc/gorpc")

@Package(Lang:"objc",Name:"com.gsrpc",Redirect:"GS")

// RPC message codes
enum Code {
    Heartbeat,WhoAmI,Request,Response,Accept,Reject,Tunnel,TunnelWhoAmI
}

enum State{
    Disconnect,Connecting,Connected,Disconnecting,Closed
}

@Flag
enum Tag{
    I8(0),I16(1),I32(2),I64(3),List(4),Table(5),String(6),Skip(7)
}

// RPC message
@gslang.POD
table Message {
    Code    Code;       // message code
    byte    Agent;      // message agent id
    byte[]  Content;    // message content
}

@gslang.POD
table Param {
    byte[] Content;
}


@gslang.POD
table Request {
    uint32      ID;
    uint16      Service;
    uint16      Method;
    Param[]     Params;
    uint64      Trace;
    uint32      Prev; // prev call stack service id
}

@gslang.POD
table Response {
    uint32      ID;
    sbyte       Exception; // exception id
    byte[]      Content;
    uint64      Trace;
}

enum OSType {
    Windows(0),Linux(1),OSX(2),WP(3),Android(4),iOS(5)
}

enum ArchType {
    X86(0),X64(1),ARM(2)
}

// The client device type
@gslang.POD
table Device {
    string      ID;             // device udid
    string      Type;           // device type
    ArchType    Arch;           // device arch type
    OSType      OS;             // device os type
    string      OSVersion;      // device os reversion
    string      AppKey;         // app key string
}

@gslang.POD
table WhoAmI {
    Device      ID;             // device name
    byte[]      Context;         // context data
}

@Exception
table InvalidContract {
}

@Exception
table UnmarshalException {
}

@Exception
table RemoteException {
}

@gslang.POD
table KV {
    byte[]     Key;
    byte[]     Value;
}

@gslang.POD
table Time {
    uint64  Second;
    uint64  Nano;
}
