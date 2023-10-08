// Just for testing stuff out. Not part of the project.

package main

import (
	"time"

	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

func testMachineGame() {
	ms := newMachineGameServer(
		core.WhiteColor,
		core.CapturesMandatory,
		core.BestNotMandatory,
		800*time.Millisecond,
		minimax.UnweightedCountHeuristic,
	)
	go ms.run()

	gc := newGameClient(ms.humanPlies, stdioRawClient())
	go gc.run()

	ms.setClient <- gc

	<-ms.ended
}
