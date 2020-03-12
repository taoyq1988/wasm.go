package aot

import (
	"github.com/zxh0/wasm.go/binary"
)

type moduleCompiler struct {
	printer
	moduleInfo
}

func (c *moduleCompiler) compile() {
	c.genModule()
	c.genDummy()
	c.genNew()
	c.println("")
	c.genMemInit()
	c.println("")
	c.genExternalFuncs()
	c.genInternalFuncs()
	c.genExportedFuncs()
	c.genGet()
	c.genGetGlobalVal()
	c.genCallFunc()
	c.genUtils()
}

func (c *moduleCompiler) genModule() {
	c.print(`// Code generated by wasm.go. DO NOT EDIT.

package main

import (
	gobin "encoding/binary"
	"math"
	"math/bits"

	"github.com/zxh0/wasm.go/binary"
	"github.com/zxh0/wasm.go/instance"
	"github.com/zxh0/wasm.go/interpreter"
)

var LE = gobin.LittleEndian

type aotModule struct {
	importedFuncs []instance.Function
	table         instance.Table
	memory        instance.Memory
	globals       []instance.Global
}
`)
}

func (c *moduleCompiler) genDummy() {
	c.print(`
// TODO
func dummy() {
	_ = bits.Add
	_ = binary.Decode
	_ = interpreter.NewInstance
}
`)
}

func (c *moduleCompiler) genNew() {
	funcCount := len(c.importedFuncs)
	globalCount := len(c.importedGlobals) + len(c.module.GlobalSec)
	c.printf(`
func Instantiate(iMap instance.Map) (instance.Instance, error) {
	m := &aotModule{
		importedFuncs: make([]instance.Function, %d),
		globals:       make([]instance.Global, %d),
	}
`, funcCount, globalCount)

	for i, imp := range c.importedFuncs {
		ft := c.module.TypeSec[imp.Desc.FuncType]
		c.printf(`	m.importedFuncs[%d] = iMap["%s"].Get("%s").(instance.Function) // %s%s`,
			i, imp.Module, imp.Name, ft.GetSignature(), "\n")
	}
	if len(c.importedTables) > 0 {
		c.printf(`	m.table = iMap["%s"].Get("%s").(instance.Table)%s`,
			c.importedTables[0].Module, c.importedTables[0].Name, "\n")
	} else if len(c.module.TableSec) > 0 {
		c.printf("	m.table = interpreter.NewTable(%d, %d)\n",
			c.module.TableSec[0].Limits.Min, c.module.TableSec[0].Limits.Max)
	}
	if len(c.importedMemories) > 0 {
		c.printf(`	m.memory = iMap["%s"].Get("%s").(instance.Memory)%s`,
			c.importedTables[0].Module, c.importedTables[0].Name, "\n")
	} else if len(c.module.MemSec) > 0 {
		c.printf("	m.memory = interpreter.NewMemory(%d, %d)\n",
			c.module.MemSec[0].Min, c.module.MemSec[0].Max)
	}
	for i, imp := range c.importedGlobals {
		c.printf(`	m.globals[%d] = iMap["%s"].Get("%s").(instance.Global)%s`,
			i, imp.Module, imp.Name, "\n")
	}
	for i, g := range c.module.GlobalSec {
		c.printf("	m.globals[%d] = interpreter.NewGlobal(%d, %t, %d)\n",
			len(c.importedGlobals)+i, g.Type.ValType, g.Type.Mut == 1, 0) // TODO
	}

	c.println("	m.initMem()")
	c.println("	return m, nil // TODO\n}")
}

func (c *moduleCompiler) genMemInit() {
	c.println("func (m *aotModule) initMem() {")
	for _, data := range c.module.DataSec {
		if len(data.Init) > 0 {
			offset := getOffset(data.Offset)
			c.printf("	m.memory.Write(%d, []byte(%q))\n",
				offset, data.Init)
		}
	}
	c.println("}")
}

func getOffset(constExpr []binary.Instruction) int {
	if len(constExpr) == 0 {
		return 0
	}
	instr := constExpr[len(constExpr)-1]
	switch instr.Opcode {
	case binary.I32Const:
		return int(instr.Args.(int32))
	default:
		panic("TODO")
	}
}

func (c *moduleCompiler) genExternalFuncs() {
	for i, imp := range c.importedFuncs {
		fc := newExternalFuncCompiler()
		ft := c.module.TypeSec[imp.Desc.FuncType]
		c.printf("// %s.%s %s\n", imp.Module, imp.Name, ft.GetSignature())
		c.println(fc.compile(i, ft))
	}
}

func (c *moduleCompiler) genInternalFuncs() {
	importedFuncCount := len(c.importedFuncs)
	for i, ftIdx := range c.module.FuncSec {
		fc := newInternalFuncCompiler(c.moduleInfo)
		fIdx := importedFuncCount + i
		ft := c.module.TypeSec[ftIdx]
		code := c.module.CodeSec[i]
		c.printf("// %s\n", ft.GetSignature())
		c.println(fc.compile(fIdx, ft, code))
	}
}

func (c *moduleCompiler) genExportedFuncs() {
	for i, exp := range c.module.ExportSec {
		if exp.Desc.Tag == binary.ExportTagFunc {
			fc := newExportedFuncCompiler(len(c.importedFuncs))
			fIdx := int(exp.Desc.Idx)
			ft := c.getFuncType(fIdx)
			c.printf("// %s %s\n", exp.Name, ft.GetSignature())
			c.println(fc.compile(i, fIdx, ft))
		}
	}
}

func (c *moduleCompiler) genGet() {
	c.print(`
func (m *aotModule) Get(name string) interface{} {
	panic("TODO")
}
`)
}
func (c *moduleCompiler) genGetGlobalVal() {
	c.print(`
func (m *aotModule) GetGlobalValue(name string) (interface{}, error) {
	panic("TODO")
}
`)
}
func (c *moduleCompiler) genCallFunc() {
	c.println("")
	c.println(`func (m *aotModule) CallFunc(name string, args ...interface{}) (interface{}, error) {`)
	c.println("	switch name {")
	for i, exp := range c.module.ExportSec {
		c.printf("	case \"%s\": return m.exported%d(args...)\n", exp.Name, i)
	}
	c.println(`	default: panic("TODO")`)
	c.println("	}")
	c.println("}")
}

func (c *moduleCompiler) genUtils() {
	c.print(`
func (m *aotModule) readU8(offset uint64) byte {
	var buf [1]byte
	m.memory.Read(offset, buf[:])
	return buf[0]
}
func (m *aotModule) readU16(offset uint64) uint16 {
	var buf [2]byte
	m.memory.Read(offset, buf[:])
	return LE.Uint16(buf[:])
}
func (m *aotModule) readU32(offset uint64) uint32 {
	var buf [4]byte
	m.memory.Read(offset, buf[:])
	return LE.Uint32(buf[:])
}
func (m *aotModule) readU64(offset uint64) uint64 {
	var buf [8]byte
	m.memory.Read(offset, buf[:])
	return LE.Uint64(buf[:])
}

func (m *aotModule) writeU8(offset uint64, n byte) {
	var buf [1]byte
	buf[0] = n
	m.memory.Write(offset, buf[:])
}
func (m *aotModule) writeU16(offset uint64, n uint16) {
	var buf [2]byte
	LE.PutUint16(buf[:], n)
	m.memory.Write(offset, buf[:])
}
func (m *aotModule) writeU32(offset uint64, n uint32) {
	var buf [4]byte
	LE.PutUint32(buf[:], n)
	m.memory.Write(offset, buf[:])
}
func (m *aotModule) writeU64(offset uint64, n uint64) {
	var buf [8]byte
	LE.PutUint64(buf[:], n)
	m.memory.Write(offset, buf[:])
}

// utils
func b2i(b bool) uint64 { if b { return 1 } else { return 0 } }
func f32(i uint64) float32 { return math.Float32frombits(uint32(i)) }
func u32(f float32) uint64 { return uint64(math.Float32bits(f)) }
func f64(i uint64) float64 { return math.Float64frombits(i) }
func u64(f float64) uint64 { return math.Float64bits(f) }
`)
}
