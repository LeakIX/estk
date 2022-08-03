package main

import (
	"github.com/LeakIX/estk/lib"
	"github.com/alecthomas/kong"
	"io/ioutil"
	"log"
	"os"
)

var App struct {
	List  lib.LsCommand   `cmd help:"List indices"`
	Dump  lib.DumpCommand `cmd help:"Dump indices"`
	Url   string          `required name:"url" help:"Base Kibana/ES url"`
	Debug bool            `short:"d" help:"Debug mode" default:"false"`
}

func main() {
	var err error
	ctx := kong.Parse(&App)
	queryDispatcher := &lib.EsQueryDispatcher{
		BaseUrl:   App.Url,
		LogOutput: os.Stderr,
	}
	if !App.Debug {
		queryDispatcher.LogOutput = ioutil.Discard
	} else {
		lib.DebugWriter = os.Stderr
		log.SetOutput(queryDispatcher.LogOutput)
	}
	err = queryDispatcher.DetectEsVersion()
	if err != nil {
		ctx.FatalIfErrorf(err)
	}
	err = ctx.Run(queryDispatcher)
	ctx.FatalIfErrorf(err)
}
