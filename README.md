# gbemu

Toy Game Boy emulator, because why not.

Built from scratch using only the following resources:
- https://gbdev.io/pandocs/
- https://gbdev.io/gb-opcodes/optables/
- https://rednex.github.io/rgbds/gbz80.7.html

Currently implements CPU instructions, interrupts, and most of the address space
(ROM, RAM, IO registers, timers).

Missing (for a rainy day):
- Finish Video processing: Most of the video processing is implemented but
    untested, and there are certainly bugs. Using something like
    https://github.com/mattcurrie/mealybug-tearoom-tests would be nice to ensure
    correctness against real hardware.
- Sound: Sound is completely mocked out right now.
- Joypad IN: The emulator window is not hooked up to input.
- Multi-platform: Currently uses a rendering framework that only works on OSX.
  Need to switch to a cross-platform solution.

## Demo

Try to run the emulator using the excellent "Is that a demo in your pocket" Demo.
https://gbhh.avivace.com/game/is-that-a-demo-in-your-pocket

## Design Choices

- Readability / Maintainability over Efficiency. The emulator does not need to
  be overly optimized since it is expected to run on hardware many times more
  powerful than the emulated hardware. Therefor, we forgo optimizations that
  would usually be seen in emulators/virtual machines to make the code easier to
  develop.
- Lean heavily on external integration tests. There are some good test ROMs for
  developing emulators. Rely primarily on these ROMs for testing, and use more
  detailed unit tests when debugging / deep diving into specific issues, or
  testing core functionality (math or bit operations).

## Development

- main.go  - main entry into application
- instruction-gen/  - logic to generate the instructions.gen.go filo from a
  3-party instruction spec file
- pkg/emulator/  - core logic

To re-generate instructions.gen.go run `go generate ./...`

To run tests do `go test ./...`