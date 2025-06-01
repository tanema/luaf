---
layout: doc
title: Writing a JIT language
---

Here are some notes and links that I want to keep track of while I learn how to write a jit


- [POC JIT on apple m1 chip that I wrote](https://github.com/tanema/go-jit-macos-arm64-poc)
- [Go ASM generation](https://go.dev/doc/asm)
- [JIT-compiler](https://github.com/bspaans/jit-compiler)
- [gojit](https://github.com/nelhage/gojit)
- [write a jit](https://medium.com/kokster/writing-a-jit-compiler-in-golang-964b61295f)
- [things I learned while writing a jit](https://www.tumblr.com/nelhagedebugsshit/84342207533/things-i-learned-writing-a-jit-in-go)
- [ARM instruction set](https://iitd-plos.github.io/col718/ref/arm-instructionset.pdf)
- [SSA](https://en.wikipedia.org/wiki/Static_single-assignment_form)
- [JIT on apple silicon](https://developer.apple.com/documentation/apple-silicon/porting-just-in-time-compilers-to-apple-silicon)
- [C repo used for some insight on how to do this in Go](https://github.com/zeusdeux/jit-example-macos-arm64)
- [JIT in C article](https://medium.com/@gamedev0909/jit-in-c-injecting-machine-code-at-runtime-1463402e6242)
- [porting JIT to apple silicon](https://developer.apple.com/documentation/apple-silicon/porting-just-in-time-compilers-to-apple-silicon?language=objc)
- [extract flat (pure) binary](https://stackoverflow.com/a/13306947)
- [Making system calls from Assembly in Mac OS X](https://filippo.io/making-system-calls-from-assembly-in-mac-os-x/)
- [arm64 syscalls](https://stackoverflow.com/questions/56985859/ios-arm64-syscalls)


## ARM
We need to generate a byte array out of uint32 so we should use `binary.LittleEndian.PutUint32(a, h)`

#### Data Processing Instructions [only executed if the condition is true]
This is the general format of data instructions in arm as a reference to write the
opcodes

|------------|------|----|-----|--------|-----|------|------|-----------|-----|
|ImO not set |31  28|    |25   |24    21|20   |19  16|15  12|11        4|3   0|
|------------|------|----|-----|--------|-----|------|------|-----------|-----|
|            | 0000 | 00 | 0   | 0000   | 0   | 0000 | 0000 | 000000000 | 000 |
|            | cond | xx | imo | opcode | set | opr1 | dst  | shift     | rm  |
|------------|------|----|-----|--------|-----|------|------|-----------|-----|

|------------|------|----|-----|--------|-----|------|------|------|----------|
|ImO set to 1| 31 28|    | 25  |24    21|20   |19  16|15  12|11   8|7        0|
|------------|------|----|-----|--------|-----|------|------|------|----------|
|            | 0000 | 00 | 1   | 0000   | 0   | 0000 | 0000 | 0000 | 00000000 |
|            | cond | xx | imm | opcode | set | opr1 | dst  | rot  |  imm     |
|------------|------|----|-----|--------|-----|------|------|------|----------|

| Field  | Description |
|--------|-------------|
| cond   | conditionally executed according to the state of the CPSR condition codes.
| xx     | not used in data processing instructions
| imo    | Immediate Operand, 0 = operand2 is a register, 1 = operand2 is an immediate value
| opcode | Which instruction to run ADD, MUL, ect.
| set    | Set condition codes 0 = do not alter condition codes, 1 = set condition codes
| opr1   | 1st operand register
| dst    | Destination register
| shift  | shift applied to rm
| rm     | 2nd operand register
| rotate | shift applied to imm
| imm    | unsigned 8 bit immediate value


## Instruction Breakdown

We have the instruction `mov x2, x1` which is represented in the byte form `0xE20301AA`

I little endian form we can see `0xAA0103E2 = 10101010000000010000001111100010`

 1 0101010  00     0  00001  000000  11111  00010
   opc      shift  N  Rm     Imm6    Rn     Rd
   ORR/MOV            X1             XZR    X2

This can be broken down to:
    sf (bit 31) = 1 → 64-bit instruction
    opc (30-29) = 01 → ORR
    N = 0
    opcode = 01010 → ORR alias for MOV
    shift = 00 (LSL)
    Rm = 00001 → X1
    imm6 = 000000
    Rn = 11111 → XZR
    Rd = 00010 → X2
