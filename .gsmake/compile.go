package tasks

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslang"
	"github.com/gsdocker/gsos/fs"
	"github.com/gsmake/gsmake"
)

func getFilePath(runner *gsmake.Runner, rootDir string, orignal string) ([]string, error) {

	var path string

	if strings.HasPrefix(orignal, "./") {
		path = filepath.Join(runner.StartDir(), orignal)
	} else {
		path = filepath.Join(rootDir, "src", orignal)

		if !fs.Exists(path) {
			return nil, gserrors.Newf(ErrUnknownPath, "unknown gslang path :%s", path)
		}
	}

	if fs.IsDir(path) {

		var files []string

		filepath.Walk(path, func(newpath string, info os.FileInfo, err error) error {

			if info.IsDir() && path != newpath {
				return filepath.SkipDir
			}

			if filepath.Ext(info.Name()) == ".gs" {
				files = append(files, newpath)
			}

			return nil
		})

		return files, nil
	}

	return []string{path}, nil
}

func compileModule(runner *gsmake.Runner, name string, files []string, codeGen gslang.CodeGen) error {

	rootDir := runner.RootFS().DomainDir("gslang")

	runner.D("gslang root dir :%s", rootDir)

	compiler := gslang.NewCompiler(name, gslang.HandleError(func(err *gslang.Error) {
		gserrors.Panicf(err.Orignal, "parse %s error\n\t%s", err.Start, err.Text)
	}))

	files = append(files, "github.com/gsdocker/gslang", "github.com/gsrpc/gsrpc")

	for _, file := range files {
		paths, err := getFilePath(runner, rootDir, file)

		if err != nil {
			return err
		}

		for _, path := range paths {
			runner.D("[%s] compile script :%s", name, path)
			if err := compiler.Compile(path); err != nil {
				return err
			}
		}
	}

	err := compiler.Link()

	if err != nil {
		return err
	}

	return compiler.Gen(codeGen)
}
