package aot

import "github.com/zxh0/wasm.go/binary"

type BlockArgs2 struct {
	binary.BlockArgs
}

type IfArgs struct {
	binary.IfArgs
}

func analyzeBr(code binary.Code) {
	for _, instr := range code.Expr {
		switch instr.Opcode {
		case binary.Block:
		case binary.Loop:
		case binary.If:
		}
	}
}

func analyzerExpr(depth uint32, expr binary.Expr) (targets []uint32) {
	for _, instr := range expr {
		switch instr.Opcode {
		case binary.Block:
			args := instr.Args.(binary.BlockArgs)
			targets = analyzerExpr(depth+1, args.Instrs)
		case binary.Loop:
			args := instr.Args.(binary.BlockArgs)
			targets = analyzerExpr(depth+1, args.Instrs)
		case binary.If:
			args := instr.Args.(binary.IfArgs)
			targets = analyzerExpr(depth+1, args.Instrs1)
			targets2 := analyzerExpr(depth+1, args.Instrs2)
			targets = append(targets, targets2...)
		case binary.Br:
			targets = []uint32{depth - instr.Args.(uint32)}
		case binary.BrIf:
			targets = []uint32{depth - instr.Args.(uint32)}
		case binary.BrTable:
			args := instr.Args.(binary.BrTableArgs)
			for _, label := range args.Labels {
				targets = append(targets, depth-label)
			}
			targets = append(targets, depth-args.Default)
		case binary.Return:
			targets = []uint32{0}
		}
	}
	return
}
