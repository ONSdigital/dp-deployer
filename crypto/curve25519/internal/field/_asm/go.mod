module asm

go 1.16

require (
	github.com/mmcloughlin/avo v0.4.0
	golang.org/x/crypto v0.0.0
)

replace golang.org/x/crypto v0.0.0 => ../../../..

// !!! what does all of the above do ?