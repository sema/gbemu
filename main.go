package main

import (
	"github.com/alecthomas/kong"
	"github.com/sema/gbemu/pkg/emulator"
)

type runCmd struct {
	SnapshotState string `help:"Snapshot emulator state at completion for debugging" type:"path"`
	BootROM       string `help:"Use boot ROM" type:"path"`

	Path string `arg name:"path" help:"Path to ROM" type:"path"`
}

func (r *runCmd) Run() error {
	e := emulator.New()
	if err := e.Run(r.Path, r.BootROM); err != nil {
		return err
	}

	if r.SnapshotState != "" {
		if err := e.Snapshot(r.SnapshotState); err != nil {
			return err
		}
	}

	return nil
}

var root struct {
	Run runCmd `cmd help:"run ROM"`
}

func main() {
	cli := kong.Parse(&root)
	err := cli.Run()
	cli.FatalIfErrorf(err)

}
