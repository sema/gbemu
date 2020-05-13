# gbemu

Toy Game Boy emulator, because why not.

Built from scratch using only the following resources:
- https://gbdev.io/pandocs/
- https://gbdev.io/gb-opcodes/optables/
- https://rednex.github.io/rgbds/gbz80.7.html


## Design Choices

- Readability / Maintainability over Efficiency. The emulator does not need to be overly optimized since it is expected to run on hardware many times more powerful than the emulated hardware. Therefor, we forgo optimizations that would usually be seen in emulators/virtual machines to make the code easier to develop.
