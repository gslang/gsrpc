package tasks

import (
	"path/filepath"

	"github.com/gsmake/gsmake"
	"github.com/gsrpc/gsrpc/gen4go"
)

// TaskResource .
func TaskResource(runner *gsmake.Runner, args ...string) error {
	return nil
}

// TaskGorpc .
func TaskGorpc(runner *gsmake.Runner, args ...string) error {

	rootDir := filepath.Join(runner.RootFS().DomainDir("golang"), "src")

	modules, err := prepareModules(runner, "gorpc", gen4go.NewCodeGen)

	if err != nil {
		return err
	}

	return modules.compile(args, rootDir)

}
