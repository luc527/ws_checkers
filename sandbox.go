// Just for testing stuff out. Not part of the project.

package main

import (
	"log"
	"time"

	"github.com/luc527/go_checkers/core"
	"github.com/luc527/go_checkers/minimax"
)

func testMachineGame() {
	ms, err := newMachineGameServer(
		core.WhiteColor,
		core.CapturesMandatory,
		core.BestNotMandatory,
		800*time.Millisecond,
		minimax.UnweightedCountHeuristic,
	)
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}
	go ms.run()

	gc := newGameClient(ms.humanPlies, stdioRawClient())
	go gc.run()

	ms.client <- gc

	<-ms.ended
}
