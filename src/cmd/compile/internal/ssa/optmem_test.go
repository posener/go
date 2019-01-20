package ssa

import (
	"testing"

	"cmd/compile/internal/types"
)

func TestOptimizeMemoryDependency(t *testing.T) {
	c := testConfig(t)
	i64 := c.config.Types.Int64

	tests := []struct {
		name        string
		fun         fun
		assertValue func(t *testing.T, fun fun, v *Value)
	}{
		{
			name: "optimize a single non conflicting store",
			fun: c.Fun("entry",
				Bloc("entry",
					Valu("start", OpInitMem, types.TypeMem, 0, nil),
					Valu("sb", OpSB, c.config.Types.Uintptr, 0, nil),
					Valu("const", OpConst32, c.config.Types.Int, 1, nil),
					Valu("addr1", OpAddr, i64.PtrTo(), 0, 0, "sb"),
					Valu("addr2", OpAddr, i64.PtrTo(), 0, 8, "sb"),
					Valu("store", OpStore, types.TypeMem, 0, i64, "addr1", "const", "start"),
					Valu("load", OpLoad, i64, 0, nil, "addr2", "store"),
					Exit("start"),
				)),
			assertValue: func(t *testing.T, fun fun, v *Value) {
				assertMemory(t, v, fun, "load", "start")
			},
		},
		{
			name: "don't optimize a single conflicting store",
			fun: c.Fun("entry",
				Bloc("entry",
					Valu("start", OpInitMem, types.TypeMem, 0, nil),
					Valu("sb", OpSB, c.config.Types.Uintptr, 0, nil),
					Valu("const", OpConst32, c.config.Types.Int, 1, nil),
					Valu("addr1", OpAddr, i64.PtrTo(), 0, 0, "sb"),
					Valu("addr2", OpAddr, i64.PtrTo(), 0, 0, "sb"),
					Valu("store", OpStore, types.TypeMem, 0, i64, "addr1", "const", "start"),
					Valu("load", OpLoad, i64, 0, nil, "addr2", "store"),
					Exit("start"),
				)),
			assertValue: func(t *testing.T, fun fun, v *Value) {
				assertMemory(t, v, fun, "load", "store")
			},
		},
		{
			name: "optimize a single load",
			fun: c.Fun("entry",
				Bloc("entry",
					Valu("start", OpInitMem, types.TypeMem, 0, nil),
					Valu("sb", OpSB, c.config.Types.Uintptr, 0, nil),
					Valu("addr1", OpAddr, i64.PtrTo(), 0, 0, "sb"),
					Valu("addr2", OpAddr, i64.PtrTo(), 0, 0, "sb"),
					Valu("load1", OpLoad, i64, 0, nil, "addr1", "start"),
					Valu("load2", OpLoad, i64, 0, nil, "addr2", "load1"),
					Exit("start"),
				)),
			assertValue: func(t *testing.T, fun fun, v *Value) {
				assertMemory(t, v, fun, "load1", "start")
				assertMemory(t, v, fun, "load2", "start")
			},
		},
		{
			name: "mix change",
			fun: c.Fun("entry",
				Bloc("entry",
					Valu("start", OpInitMem, types.TypeMem, 0, nil),
					Valu("sb", OpSB, c.config.Types.Uintptr, 0, nil),
					Valu("const", OpConst32, c.config.Types.Int, 1, nil),
					Valu("addr1", OpAddr, i64.PtrTo(), 0, 0, "sb"),
					Valu("addr2", OpAddr, i64.PtrTo(), 0, 8, "sb"),
					Valu("addr3", OpAddr, i64.PtrTo(), 0, 16, "sb"),
					Valu("addr4", OpAddr, i64.PtrTo(), 0, 24, "sb"),
					Valu("store1", OpStore, types.TypeMem, 0, i64, "addr1", "const", "start"),
					Valu("store2", OpStore, types.TypeMem, 0, i64, "addr2", "const", "store1"),
					Valu("load1", OpLoad, i64, 0, nil, "addr3", "store2"),
					Valu("load2", OpLoad, i64, 0, nil, "addr4", "store1"),
					Exit("start"),
				)),
			assertValue: func(t *testing.T, fun fun, v *Value) {
				assertMemory(t, v, fun, "load1", "start")
				assertMemory(t, v, fun, "load2", "start")
			},
		},
		{
			name: "block separation prevents optimization",
			fun: c.Fun("entry",
				Bloc("entry",
					Valu("start", OpInitMem, types.TypeMem, 0, nil),
					Valu("sb", OpSB, c.config.Types.Uintptr, 0, nil),
					Valu("const", OpConst32, c.config.Types.Int, 1, nil),
					Valu("addr1", OpAddr, i64.PtrTo(), 0, 0, "sb"),
					Valu("store", OpStore, types.TypeMem, 0, i64, "addr1", "const", "start"),
					Goto("b2"),
				),
				Bloc("b2",
					Valu("addr2", OpAddr, i64.PtrTo(), 0, 8, "sb"),
					Valu("load", OpLoad, i64, 0, nil, "addr2", "store"),
					Exit("start"),
				)),
			assertValue: func(t *testing.T, fun fun, v *Value) {
				assertMemory(t, v, fun, "load", "store")
			},
		},
		{
			// Check that if a load value is depended a prior load. the optimization of the
			// prior load happens only after the optimization of the second one is calculated.
			name: "late optimization check",
			fun: c.Fun("entry",
				Bloc("entry",
					Valu("start", OpInitMem, types.TypeMem, 0, nil),
					Valu("sb", OpSB, c.config.Types.Uintptr, 0, nil),
					Valu("const", OpConst32, c.config.Types.Int, 1, nil),
					Valu("addr1", OpAddr, i64.PtrTo(), 0, 0, "sb"),
					Valu("addr2", OpAddr, i64.PtrTo(), 0, 8, "sb"),
					Valu("store", OpStore, types.TypeMem, 0, i64, "addr1", "const", "start"),
					// load1 has a different address than store, so it can be optimized to init mem.
					Valu("load1", OpLoad, i64, 0, nil, "addr2", "store"),
					// load2 has the same address as store, so it can be optimized only to store.
					Valu("load2", OpLoad, i64, 0, nil, "addr1", "load1"),
					Exit("start"),
				)),
			assertValue: func(t *testing.T, fun fun, v *Value) {
				assertMemory(t, v, fun, "load1", "start")
				assertMemory(t, v, fun, "load2", "store")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CheckFunc(tt.fun.f)
			optimizeMemoryDependency(tt.fun.f)
			CheckFunc(tt.fun.f)
			for _, b := range tt.fun.f.Blocks {
				for _, v := range b.Values {
					tt.assertValue(t, tt.fun, v)
				}
			}
		})
	}
}

func assertMemory(t *testing.T, v *Value, fun fun, name string, memory string) {
	t.Helper()
	if v != fun.values[name] {
		return
	}
	mem := v.Args[len(v.Args)-1]
	if got, want := mem, fun.values[memory]; got != want {
		t.Errorf("Load memory %s = %v, want %v", v, got, want)
	}
}
