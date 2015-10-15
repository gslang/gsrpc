package tasks

import (
	"github.com/gsmake/gsmake"
	"github.com/gsrpc/gsrpc/gen4objc"
)

const (
	objcDefaultRootDIR = "src"
)

// TaskObjrpc .
func TaskObjrpc(runner *gsmake.Runner, args ...string) error {

	modules, err := prepareModules(runner, "objrpc", gen4objc.NewCodeGen)

	if err != nil {
		return err
	}

	return modules.compile(args, objcDefaultRootDIR)
}
