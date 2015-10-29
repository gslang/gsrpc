package com.gsrpc;

using com.gsrpc.Device;
using com.gsrpc.Message;

@gslang.POD
table Tunnel {
    Device      ID;
    Message     Message;
}

// Named service description
@gslang.POD
table NamedService {
    string          Name;       // service name
    uint16          DispatchID; // service dispatcher id
    uint32          VNodes;     // service virtual node counter
    string          NodeName;   // service node name
}

@gslang.POD
table TunnelWhoAmI {
    NamedService[]  Services;
}
