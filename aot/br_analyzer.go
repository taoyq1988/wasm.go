package aot

import "github.com/zxh0/wasm.go/binary"

func analyzeBr(code binary.Code) {
	analyzerExpr(0, code.Expr)
}

func analyzerExpr(depth uint32, expr binary.Expr) (targets []uint32) {
	for i, instr := range expr {
		switch instr.Opcode {
		case binary.Block, binary.Loop:
			args := instr.Args.(binary.BlockArgs)
			targets = analyzerExpr(depth+1, args.Instrs)
			for _, target := range targets {
				if target == depth+1 {
					args.Instrs = append(args.Instrs, binary.Instruction{Opcode: 0xFF})
					expr[i].Args = args // hack!
					break
				}
			}
		case binary.If:
			args := instr.Args.(binary.IfArgs)
			targets = analyzerExpr(depth+1, args.Instrs1)
			targets2 := analyzerExpr(depth+1, args.Instrs2)
			targets = append(targets, targets2...)
			for _, target := range targets {
				if target == depth+1 {
					args.Instrs1 = append(args.Instrs1, binary.Instruction{Opcode: 0xFF})
					expr[i].Args = args // hack!
					break
				}
			}
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
