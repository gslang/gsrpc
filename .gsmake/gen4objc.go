package tasks

import (
	"github.com/gsdocker/gserrors"
	"github.com/gsmake/gsmake"
	"github.com/gsmake/gsmake/property"
	"github.com/gsrpc/gsrpc/gen4objc"
)

const (
	objcDefaultRootDIR = "src"
)

// TaskObjrpc .
func TaskObjrpc(runner *gsmake.Runner, args ...string) error {

	var modules map[string][]string

	err := runner.Property("gslang", runner.Name(), "gsrpc", &modules)

	if err != nil {

		if property.NotFound(err) {
			return nil
		}

		return err
	}

	var outputs map[string]string

	err = runner.Property("gslang", runner.Name(), "objrpc", &outputs)

	if err != nil {

		if !property.NotFound(err) {
			return err
		}

		outputs = make(map[string]string)
	}

	if len(args) != 0 {
		for _, name := range args {
			runner.I("[gsrpc-go] generate module :%s", name)

			module, ok := modules[name]

			if !ok {
				return gserrors.Newf(ErrModuleNotFound, "module(%s) not found", name)
			}

			rootDir := objcDefaultRootDIR

			if output, ok := outputs[name]; ok {
				rootDir = output
			}

			codegen, err := gen4objc.NewCodeGen(rootDir)

			if err != nil {
				return err
			}

			err = compileModule(runner, name, module, codegen)

			if err != nil {
				return err
			}
		}

		return nil
	}

	for name, module := range modules {

		runner.I("[gsrpc-go] generate module :%s", name)

		rootDir := objcDefaultRootDIR

		if output, ok := outputs[name]; ok {
			rootDir = output
		}

		codegen, err := gen4objc.NewCodeGen(rootDir)

		if err != nil {
			return err
		}

		err = compileModule(runner, name, module, codegen)

		if err != nil {
			return err
		}

		runner.I("[gsrpc-go] generate module :%s -- success", name)
	}

	return nil
}
