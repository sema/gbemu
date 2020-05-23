// main generates specs for each GB instruction that are embeddable in the emulator.
//
// The specs originate from https://gbdev.io/gb-opcodes/optables/, and are
// processed slightly to make the emulator logic simpler. See the `instruction`
// and `operand` structs in pkg/emulator for the semantics of the generated
// specs.
//
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
)

const outputTemplate = `//go:generate go run ../../instruction-gen/main.go ../../instruction-gen/spec.json ./instructions.gen.go{{ printf "\n" }}//go:generate go fmt ./instructions.gen.go

// GENERATED FILE - Run "go generate ./..." to update

package emulator

var instructions = []instruction{
{{ range .Unprefixed -}}
	{{ printf "  " }}{
		Opcode: "{{ .Opcode }}",
		Mnemonic:  "{{ .Mnemonic }}",
		Size:     {{ .Bytes }},
		Cycles:   []int{
			{{- range .Cycles}}{{ . }},{{ printf "\n" }}{{ end -}}
		},
		Operands:  []operand{
			{{ range .Operands -}}
			{
				Name:      "{{ .Name }}",
				Type:      {{ .Type }},
				Ref: "{{ .Ref }}",
				{{ if .RefRegister8 }}RefRegister8:  {{ .RefRegister8 }},{{ printf "\n" }}{{ end -}}
				{{- if .RefRegister16 }}RefRegister16: {{ .RefRegister16 }},{{ printf "\n" }}{{ end -}}
				{{- if .RefFlag }}RefFlag: {{ .RefFlag }},{{ printf "\n" }}{{ end -}}
				{{- if .RefFlagNegate }}RefFlagNegate: {{ .RefFlagNegate }},{{ printf "\n" }}{{ end -}}
				{{- if .RefConst8 }}RefConst8: {{ .RefConst8 }},{{ printf "\n" }}{{ end -}}
				{{- if .Increment }}IncrementReg16: {{ .Increment }},{{ printf "\n" }}{{ end -}}
				{{- if .Decrement }}DecrementReg16: {{ .Decrement }},{{ printf "\n" }}{{ end -}}
			},{{ printf "\n" }}
			{{- end }}
		},
		Flags:     flags{
			Z: "{{ .Flags.Z }}",
			N: "{{ .Flags.N }}",
			H: "{{ .Flags.H }}",
			C: "{{ .Flags.C }}",
		},
		Todo: "{{ .Todo }}",
	},{{ printf "\n" }}
{{- end }}}

var cbInstructions = []instruction{
	{{ range .Cbprefixed -}}
		{{ printf "  " }}{
			Opcode: "{{ .Opcode }}",
			Mnemonic:  "{{ .Mnemonic }}",
			Size:     {{ .Bytes }},
			Cycles:   []int{
				{{- range .Cycles}}{{ . }},{{ printf "\n" }}{{ end -}}
			},
			Operands:  []operand{
				{{ range .Operands -}}
				{
					Name:      "{{ .Name }}",
					Type:      {{ .Type }},
					Ref: "{{ .Ref }}",
					{{ if .RefRegister8 }}RefRegister8:  {{ .RefRegister8 }},{{ printf "\n" }}{{ end -}}
					{{- if .RefRegister16 }}RefRegister16: {{ .RefRegister16 }},{{ printf "\n" }}{{ end -}}
					{{- if .RefFlag }}RefFlag: {{ .RefFlag }},{{ printf "\n" }}{{ end -}}
					{{- if .RefFlagNegate }}RefFlagNegate: {{ .RefFlagNegate }},{{ printf "\n" }}{{ end -}}
					{{- if .RefConst8 }}RefConst8: {{ .RefConst8 }},{{ printf "\n" }}{{ end -}}
					{{- if .Increment }}IncrementReg16: {{ .Increment }},{{ printf "\n" }}{{ end -}}
					{{- if .Decrement }}DecrementReg16: {{ .Decrement }},{{ printf "\n" }}{{ end -}}
				},{{ printf "\n" }}
				{{- end }}
			},
			Flags:     flags{
				Z: "{{ .Flags.Z }}",
				N: "{{ .Flags.N }}",
				H: "{{ .Flags.H }}",
				C: "{{ .Flags.C }}",
			},
			Todo: "{{ .Todo }}",
		},{{ printf "\n" }}
	{{- end }}}

`

var typeToRWBits = map[string]uint{
	"operandD8":       8,
	"operandD16":      16,
	"operandA8":       16,
	"operandA8Ptr":    8,
	"operandA16":      16,
	"operandA16Ptr":   8,
	"operandR8":       8,
	"operandFlag":     1,
	"operandReg8":     8,
	"operandReg8Ptr":  8,
	"operandReg16":    16,
	"operandReg16Ptr": 8,
	"operandConst8":   0,
}

type root struct {
	Unprefixed map[string]*instruction
	Cbprefixed map[string]*instruction
}

type instruction struct {
	Opcode    string
	Mnemonic  string
	Bytes     int
	Cycles    []int
	Operands  []*operand
	Immediate bool
	Flags     flags

	// Todo flags instruction as unsupported temporarily as we expand codegen
	Todo string
}

type operand struct {
	Name string
	Type string

	Ref           string
	RefConst8     uint8
	RefRegister8  string
	RefRegister16 string
	RefFlag       string
	RefFlagNegate bool

	// R/W Bits - 1, 8 or 16 depending on the number of bits that are read or
	// written from the operand
	RWBits uint

	Bytes     int
	Immediate bool
	Increment bool
	Decrement bool
}

type flags struct {
	Z string
	N string
	H string
	C string
}

func main() {
	if len(os.Args) < 3 {
		log.Printf("Usage: %s spec.json output.go", os.Args[0])
		os.Exit(1)
	}

	specPath := os.Args[1]
	outputPath := os.Args[2]

	if !strings.HasSuffix(outputPath, ".go") {
		log.Println("Expected output file to have a .go extension")
		os.Exit(1)
	}

	log.Printf("Generating instruction implementations")
	log.Printf("Spec: %s", specPath)
	log.Printf("Output: %s", outputPath)

	err := generate(specPath, outputPath)
	if err != nil {
		log.Panic(err)
	}

	log.Println("Done")
}

func generate(specPath, outputPath string) error {
	instrucitonSpecRaw, err := ioutil.ReadFile(specPath)
	if err != nil {
		return err
	}

	var instructionSpec root
	if err := json.Unmarshal(instrucitonSpecRaw, &instructionSpec); err != nil {
		return err
	}

	postprocessSpec(&instructionSpec)

	tmpl, err := template.New("output").Parse(outputTemplate)
	if err != nil {
		return err
	}

	fp, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer fp.Close()

	log.Printf("Found %d instructions", len(instructionSpec.Unprefixed))
	return tmpl.Execute(fp, instructionSpec)
}

// postprocessSpec makes adjustments to the upstream spec to simplify
// consumption in the emulator
func postprocessSpec(instructionSpec *root) {
	for opcode, inst := range instructionSpec.Unprefixed {
		postprocessInstruction(opcode, inst, false)
	}
	for opcode, inst := range instructionSpec.Cbprefixed {
		postprocessInstruction(opcode, inst, true)
	}
}

func postprocessInstruction(opcode string, inst *instruction, isPrefixed bool) {
	inst.Opcode = opcode
	if isPrefixed {
		inst.Opcode = fmt.Sprintf("*%s", opcode)
	}

	if inst.Opcode == "0xF8" {
		// LD HL SP+r8 is specified with a misleading notation in the spec file
		// (SP+r8 is noted as just SP+ which has a different meaning)
		inst.Operands = []*operand{
			&operand{Name: "HL", Immediate: true},
			&operand{Name: "SP", Immediate: true},
			&operand{Name: "r8", Immediate: true},
		}
	}

	if inst.Mnemonic == "XOR" || inst.Mnemonic == "AND" || inst.Mnemonic == "OR" || inst.Mnemonic == "CP" || inst.Mnemonic == "SUB" {
		// 8bit logical and arithmetic instructions take two arguments, A and X (X=reg8|reg16Ptr).
		// The spec does not include the implicit A argument. Adding the argument to
		// make the emulator logic simpler.
		inst.Operands = []*operand{
			&operand{Name: "A", Immediate: true},
			inst.Operands[0],
		}
	}

	if inst.Mnemonic == "RLA" || inst.Mnemonic == "RLCA" || inst.Mnemonic == "RRA" || inst.Mnemonic == "RRCA" || inst.Mnemonic == "DAA" || inst.Mnemonic == "CPL" {
		// Instructions that take A as an operand, but does not declare so in the spec
		inst.Operands = []*operand{
			&operand{Name: "A", Immediate: true},
		}
	}

	if (inst.Mnemonic == "JP" || inst.Mnemonic == "JR" || inst.Mnemonic == "CALL") && len(inst.Operands) == 2 {
		// Swap the order of operands, such that operand-0 is always the
		// destination and the second is an (optional) condition for the
		// jump. This simplifies the emulator logic.
		inst.Operands = []*operand{inst.Operands[1], inst.Operands[0]}
	}

	if len(inst.Cycles) == 2 {
		// Simplify emulator logic by listing the "action taken" cycle count last rather than first
		inst.Cycles = []int{inst.Cycles[1], inst.Cycles[0]}
	}

	// Record cycles in "machine cycles" and not "clock cycles"
	var scaledCycles []int
	for _, cycle := range inst.Cycles {
		scaledCycles = append(scaledCycles, cycle/4)
	}
	inst.Cycles = scaledCycles

	if strings.HasPrefix(inst.Mnemonic, "ILLEGAL") {
		// Illegal instructions have mnemonics on the format
		// ILLEGAL_{OPCODE}, which makes them difficult to switch on
		// in the template. Normalize these into ILLEGAL to fit the format
		// of other mnemonics.
		inst.Mnemonic = "ILLEGAL"
	}

	if inst.Mnemonic == "LDH" {
		// LDH is equivalent with LD, with the only difference being that LDH accepts a8 (FF00 offset)
		// operands. Due to the way we implement LD, it is trivial to also support the LDH case at the
		// same time.
		inst.Mnemonic = "LD"
	}

	// The C flag and C register are used interchangably in the spec file. Use this hint to determine
	// when to interpret C as a flag and not a register.
	takesFlagNotRegister := inst.Mnemonic == "CALL" || inst.Mnemonic == "RET" || inst.Mnemonic == "JP" || inst.Mnemonic == "JR"

	for _, op := range inst.Operands {
		// Infer a "type" for each operand, e.g. to differentiate between
		// data following operands (d8, d16), registers (reg8, reg16), flags (flag),
		// etc. The Ref* fields are used depending on the type to further specify
		// what the operand references (e.g. the exact flag).
		//
		if len(op.Name) == 1 && strings.Contains("ABCDEHL", op.Name) && !takesFlagNotRegister {
			op.Type = "operandReg8"
			op.RefRegister8 = fmt.Sprintf("register%s", op.Name)
		} else if op.Name == "AF" || op.Name == "BC" || op.Name == "DE" || op.Name == "HL" || op.Name == "SP" {
			op.Type = "operandReg16"
			op.RefRegister16 = fmt.Sprintf("register%s", op.Name)
		} else if len(op.Name) == 1 && strings.Contains("ZNHC", op.Name) && takesFlagNotRegister {
			op.Type = "operandFlag"
			op.RefFlag = fmt.Sprintf("flag%s", op.Name)
		} else if op.Name == "NC" || op.Name == "NZ" {
			op.Type = "operandFlag"
			op.RefFlag = fmt.Sprintf("flag%s", op.Name[1:])
			op.RefFlagNegate = true
		} else if strings.HasSuffix(op.Name, "H") {
			op.Type = "operandConst8"
			if c, err := hex.DecodeString(op.Name[0:2]); err != nil {
				log.Panicf("unable to convert hex value \"%s\" to uint8: %s", op.Name, err)
			} else {
				op.RefConst8 = uint8(c[0])
			}
		} else if len(op.Name) == 1 && strings.Contains("01234567", op.Name) {
			op.Type = "operandConst8"
			if c, err := strconv.Atoi(op.Name); err != nil {
				log.Panicf("unable to convert const \"%s\" to uint8: %s", op.Name, err)
			} else {
				op.RefConst8 = uint8(c)
			}
		} else if op.Name == "d8" || op.Name == "d16" || op.Name == "a8" || op.Name == "a16" || op.Name == "r8" {
			op.Type = fmt.Sprintf("operand%s", strings.Title(op.Name))
		} else {
			log.Panicf("unable to determine type of operand: %s", op.Name)
		}

		if op.Increment {
			op.Name = fmt.Sprintf("%s+", op.Name)
		} else if op.Decrement {
			op.Name = fmt.Sprintf("%s-", op.Name)
		}
		if !op.Immediate {
			op.Type = fmt.Sprintf("%sPtr", op.Type)
			op.Name = fmt.Sprintf("(%s)", op.Name)
		}

		rwBits, ok := typeToRWBits[op.Type]
		if !ok {
			log.Panicf("unexpected type when resolving RWBits: %s", op.Type)
		}
		op.RWBits = rwBits
	}

	if inst.Opcode == "0xF8" {
		// LD HL SP+r8 is different from all other LD(8/16) instructions as it modifies
		// flags. Separate it out into a seprate Mnemonic.
		inst.Mnemonic = "LDSP"
	} else if inst.Mnemonic == "ADD" && inst.Operands[0].RefRegister16 == "registerSP" {
		// ADDSP is different from ADD8 and ADD16, as we add a signed value to the current SP
		inst.Mnemonic = "ADDSP"
	} else if inst.Mnemonic == "LD" || inst.Mnemonic == "INC" || inst.Mnemonic == "DEC" || inst.Mnemonic == "ADD" {
		// Differentiate between 8bit and 16bit instructions, as they
		// difference between the amount of data they expect to read and write
		//
		// Note, we use the "size" of the last operand to determine the value we operate on, as the
		// spec sometimes uses [a16] as a 8bit destination, and sometimes as a 16bit destination in LD
		// instructions.
		inst.Mnemonic = fmt.Sprintf("%s%d", inst.Mnemonic, inst.Operands[len(inst.Operands)-1].RWBits)
	}

	if inst.Flags.C != "-" || inst.Flags.H != "-" || inst.Flags.N != "-" || inst.Flags.Z != "-" {
		if inst.Mnemonic != "INC8" && inst.Mnemonic != "DEC8" && inst.Mnemonic != "XOR" && inst.Mnemonic != "AND" && inst.Mnemonic != "OR" && inst.Mnemonic != "BIT" && inst.Mnemonic != "RL" && inst.Mnemonic != "RLA" && inst.Mnemonic != "RLC" && inst.Mnemonic != "RLCA" && inst.Mnemonic != "RR" && inst.Mnemonic != "RRA" && inst.Mnemonic != "RRCA" && inst.Mnemonic != "SLA" && inst.Mnemonic != "SRA" && inst.Mnemonic != "SRL" && inst.Mnemonic != "CP" && inst.Mnemonic != "SUB" && inst.Mnemonic != "ADD8" && inst.Mnemonic != "SCF" && inst.Mnemonic != "CCF" && inst.Mnemonic != "SWAP" && inst.Mnemonic != "POP" && inst.Mnemonic != "ADC" && inst.Mnemonic != "SBC" && inst.Mnemonic != "ADD16" && inst.Mnemonic != "ADDSP" && inst.Mnemonic != "DAA" && inst.Mnemonic != "CPL" && inst.Mnemonic != "RRC" && inst.Mnemonic != "LDSP" {
			inst.Todo = "mutates flags"
		}
	}
}
