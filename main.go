package main

import (
	"log"

	"github.com/sema/gbemu/pkg/emulator"
)

func main() {
	romPath := "gb-test-roms/cpu_instrs/individual/01-special.gb"

	e := emulator.New()
	if err := e.Run(romPath); err != nil {
		log.Fatal(err)
	}

}
