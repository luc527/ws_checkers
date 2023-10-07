package main

import "github.com/luc527/go_checkers/core"

// TODO move copy functions to core package

func copyPly(p core.Ply) core.Ply {
	c := make([]core.Instruction, len(p))
	copy(c, p)
	return c
}

func copyPlies(ps []core.Ply) []core.Ply {
	c := make([]core.Ply, len(ps))
	for i, p := range ps {
		c[i] = copyPly(p)
	}
	return c
}
