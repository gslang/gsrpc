package com.gsrpc.trace;

using com.gsrpc.KV;
using com.gsrpc.Time;
using gslang.Package;

@Package(Lang:"golang",Name:"com.gsrpc.trace",Redirect:"github.com/gsrpc/gorpc/trace")


table EvtRPC {
    uint64     Trace;       // trace id
    uint32     ID;          // the rpc call id
    uint32     Prev;    // parent span
    string     Probe;       // probe address
    Time       StartTime;   // span start timestamp
    Time       EndTime;     // span end timestamp
    KV[]       Attributes;  // customer attributes
}
