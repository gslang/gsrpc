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
		path, _ = filepath.Abs(filepath.Join(runner.StartDir(), orignal))
	} else {
		path, _ = filepath.Abs(filepath.Join(rootDir, "src", orignal))

		if !fs.Exists(path) {
			return nil, gserrors.Newf(ErrUnknownPath, "unknown gslang path :%s", path)
		}
	}

	fi, err := os.Stat(path)

	if err == nil && fi.IsDir() {

		var files []string

		if path, err = filepath.EvalSymlinks(path); err != nil {
			return files, gserrors.Newf(err, "try search *.gs files error")
		}

		err = filepath.Walk(path, func(newpath string, info os.FileInfo, err error) error {

			if info.IsDir() && path != newpath {

				return filepath.SkipDir
			}

			if filepath.Ext(info.Name()) == ".gs" {

				files = append(files, newpath)
			}

			return err
		})

		return files, err
	}

	return []string{path}, err
}

func compileModule(runner *gsmake.Runner, name string, files []string, codeGen gslang.Visitor) error {

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

	return compiler.Visit(codeGen)
}
