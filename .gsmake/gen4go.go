package tasks

import (
	"path/filepath"

	"github.com/gsdocker/gserrors"
	"github.com/gsmake/gsmake"
	"github.com/gsmake/gsmake/property"
	"github.com/gsrpc/gsrpc/gen4go"
)

// TaskGorpc .
func TaskGorpc(runner *gsmake.Runner, args ...string) error {

	var modules map[string][]string

	err := runner.Property("gslang", runner.Name(), "gsrpc", &modules)

	if err != nil && !property.NotFound(err) {
		return err
	}

	rootDir := filepath.Join(runner.RootFS().DomainDir("golang"), "src")

	codegen, err := gen4go.NewCodeGen(rootDir)

	if err != nil {
		return err
	}

	if len(args) != 0 {
		for _, name := range args {
			runner.I("[gsrpc-go] generate module :%s", name)

			module, ok := modules[name]

			if !ok {
				return gserrors.Newf(ErrModuleNotFound, "module(%s) not found", name)
			}

			err := compileModule(runner, name, module, codegen)

			if err != nil {
				return err
			}
		}

		return nil
	}

	for name, module := range modules {

		runner.I("[gsrpc-go] generate module :%s", name)

		err := compileModule(runner, name, module, codegen)

		if err != nil {
			return err
		}

		runner.I("[gsrpc-go] generate module :%s -- success", name)
	}

	return nil
}
