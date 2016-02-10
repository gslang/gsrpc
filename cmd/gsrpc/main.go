package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gslang"
	"github.com/gsrpc/gsrpc/gen4go"
	"github.com/gsrpc/gsrpc/gen4java"
	"github.com/gsrpc/gsrpc/gen4objc"
)

var lang = flag.String("lang", "golang", "gsrpc generate language")
var output = flag.String("o", ".", "gsrpc output directory")

var langs = map[string]func(rootpath string, skips []string) (gslang.Visitor, error){
	"golang": gen4go.NewCodeGen,
	"java":   gen4java.NewCodeGen,
	"objc":   gen4objc.NewCodeGen,
}

func main() {
	gslogger.Console("$content", "2006-01-02 15:04:05.999999")
	gslogger.NewFlags(gslogger.ERROR | gslogger.WARN | gslogger.INFO)
	log := gslogger.Get("gsrpc")

	defer func() {
		if e := recover(); e != nil {
			log.E("%s", e)
		}

		gslogger.Join()
	}()

	flag.Parse()

	*output, _ = filepath.Abs(*output)

	log.I("Start gsRPC With Target Language(%s)", *lang)

	codegenF, ok := langs[*lang]

	if !ok {
		log.E("unknown gsrpc object language :%s", *lang)
		os.Exit(1)
	}

	codegen, err := codegenF(*output, []string{"github.com/gsrpc/gslang"})

	if err != nil {
		gserrors.Panicf(err, "create language(%s) codegen error", *lang)
	}

	compiler := gslang.NewCompiler("gsrpc", gslang.HandleError(func(err *gslang.Error) {
		gserrors.Panicf(err.Orignal, "parse %s error\n\t%s", err.Start, err.Text)
	}))

	for _, file := range flag.Args() {
		log.I("Compile gsLang File :%s", file)
		if err := compiler.Compile(file); err != nil {
			gserrors.Panicf(err, "compile %s error", file)
		}
	}

	log.I("Link ...")
	err = compiler.Link()

	if err != nil {
		gserrors.Panicf(err, "link error")
	}

	log.I("Output Directory :%s", *output)

	if err := compiler.Visit(codegen); err != nil {
		gserrors.Panicf(err, "generate language codes(%s) error", *lang)
	}

	log.I("Run gsRPC Compile -- Success")
}
