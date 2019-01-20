package ssa

import (
	"log"

	"cmd/compile/internal/types"
)

// optimizeMemoryDependency tries to modify load operations memory argument for
// earlier values, which are not creating any memory conflicts.
func optimizeMemoryDependency(f *Func) {
	for i := len(f.Blocks) - 1; i >= 0; i-- {
		b := f.Blocks[i]
		for j := len(b.Values) - 1; j >= 0; j-- {
			v := b.Values[j]
			// Only optimize load values.
			if v.Op != OpLoad {
				continue
			}
			if m := findOptMem(v); m != nil && m != memoryArg(v) {
				f.Logf("Optimizing memory of load %s -> %s\n", v.LongString(), m)
				v.SetArg(1, m)
			}
		}
	}
}

// findOptMem returns an optimal value that the load memory argument can have
// without any memory conflicts.
func findOptMem(load *Value) *Value {
	var optMem *Value
	// Iterate over a memory chain of load and store operations. In each
	// iteration, v points to a value, and m points to the value's memory
	// argument.
	for v, m := load, memoryArg(load); m != nil; v, m = m, memoryArg(m) {
		// For memory modifying operation, check that the address read by load,
		// is not modified by v.
		if v.Op == OpStore && valueJoint(load, v) {
			break
		}

		// Keep optimization only to block level.
		if load.Block.ID != m.Block.ID {
			break
		}

		// The tested value, does not conflict with this save operation,
		// hence it can point to the next value in the memory chain.
		optMem = m
	}
	return optMem
}

// memoryArg returns the memory argument for load and store operations.
// For other operations it returns nil.
func memoryArg(v *Value) *Value {
	switch v.Op {
	case OpLoad, OpStore:
		return v.Args[len(v.Args)-1]
	default:
		return nil
	}
}

// valueJoint tests if two values share the same memory address.
func valueJoint(v1 *Value, v2 *Value) bool {
	a1, s1 := addrSize(v1)
	a2, s2 := addrSize(v2)
	return !disjoint(a1, s1, a2, s2)
}

// addrSize returns address and size of load and store values.
func addrSize(v *Value) (*Value, int64) {
	switch v.Op {
	case OpLoad:
		return v.Args[0], v.Type.Size()
	case OpStore:
		return v.Args[0], v.Aux.(*types.Type).Size()
	default:
		log.Fatalf("Operation %s was not expected", v.Op)
		return nil, 0
	}
}
