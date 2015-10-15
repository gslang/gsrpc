package tasks

import (
	"github.com/gsdocker/gserrors"
	"github.com/gsmake/gsmake"
	"github.com/gsmake/gsmake/property"
	"github.com/gsrpc/gslang"
)

// Config .
type Config struct {
	Output string   // output path
	Skips  []string // skip list
}

// _Modules gsrpc compile module
type _Modules struct {
	runner  *gsmake.Runner      // current gsmake runner
	modules map[string][]string // compile modules
	configs map[string]Config   // generate config
	codegen CodeGenF            //codgen factory
}

// CodeGenF .
type CodeGenF func(rootdir string, skips []string) (gslang.Visitor, error)

func prepareModules(runner *gsmake.Runner, lang string, codegen CodeGenF) (*_Modules, error) {

	modules := &_Modules{
		runner:  runner,
		codegen: codegen,
	}

	err := runner.Property("gslang", runner.Name(), "gsrpc", &modules.modules)

	if err != nil {

		if property.NotFound(err) {
			return modules, nil
		}

		return nil, err
	}

	err = runner.Property("gslang", runner.Name(), lang, &modules.configs)

	if err != nil {

		if !property.NotFound(err) {
			return nil, err
		}

		modules.configs = make(map[string]Config)
	}

	return modules, nil
}

func (modules *_Modules) compileModule(name string, files []string, defaultpath string) error {

	rootDir := defaultpath

	var skips []string

	if config, ok := modules.configs[name]; ok {

		if config.Output != "" {
			rootDir = config.Output
		}

		skips = config.Skips
	}

	codegen, err := modules.codegen(rootDir, skips)

	if err != nil {
		return err
	}

	err = compileModule(modules.runner, name, files, codegen)

	if err != nil {
		return err
	}

	return nil
}

func (modules *_Modules) compile(args []string, defaultpath string) error {

	if len(args) != 0 {
		for _, name := range args {
			modules.runner.I("[gsrpc-go] generate module :%s", name)

			module, ok := modules.modules[name]

			if !ok {
				return gserrors.Newf(ErrModuleNotFound, "module(%s) not found", name)
			}

			if err := modules.compileModule(name, module, defaultpath); err != nil {
				return err
			}
		}

		return nil
	}

	for name, module := range modules.modules {

		if err := modules.compileModule(name, module, defaultpath); err != nil {
			return err
		}
	}

	return nil

}
