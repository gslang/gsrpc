package tasks

import (
	"github.com/gsmake/gsmake"
	"github.com/gsrpc/gsrpc/gen4java"
)

const (
	jvmDefaultRootDIR = "src/main/java"
)

// TaskJvmrpc .
func TaskJvmrpc(runner *gsmake.Runner, args ...string) error {

	modules, err := prepareModules(runner, "jvmrpc", gen4java.NewCodeGen)

	if err != nil {
		return err
	}

	return modules.compile(args, jvmDefaultRootDIR)
}
