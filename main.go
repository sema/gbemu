package main

import (
	"github.com/alecthomas/kong"
	"github.com/sema/gbemu/pkg/emulator"
)

type runCmd struct {
	//Recursive bool `help:"Recursively remove files."`

	Path string `arg name:"path" help:"Path to ROM" type:"path"`
}

func (r *runCmd) Run() error {
	e := emulator.New()
	return e.Run(r.Path)
}

var root struct {
	Run runCmd `cmd help:"run ROM"`
}

func main() {
	cli := kong.Parse(&root)
	err := cli.Run()
	cli.FatalIfErrorf(err)

}
