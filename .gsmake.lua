name "github.com/gsrpc/gsrpc"

plugin "github.com/gsmake/golang"


properties.golang = {
    dependencies = {
        { name = "github.com/gsrpc/gslang" };
    };

    binaries = { "cmd/gsrpc" };
}
