package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/luc527/go_checkers/core"
)

func gameMonitorStuff() {
	gm := newGameMonitor(core.CapturesMandatory, core.BestNotMandatory)
	go gm.run()

	done := make(chan struct{})
	{
		l := make(chan gameState)
		gm.addListener <- l
		go func() {
			for s := range l {
				if s.result.Over() {
					fmt.Println("Whew! The game is finally over.")
					close(done)
				}
			}
		}()
	}

	{
		l := make(chan gameState)
		gm.addListener <- l
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				s, ok := <-l
				if !ok {
					break
				}

				fmt.Printf("Board:\n%v\n", s.board.String())
				fmt.Printf("Play: %v\n", s.toPlay)
				fmt.Printf("Result: %v\n", s.result.String())

				fmt.Println("Hmm... Let me think...")
				<-ticker.C

				c := make(chan bool)
				gm.doPly <- plyMessage{
					i: rand.Intn(len(s.plies)),
					v: s.v,
					c: c,
				}
				go func() {
					if ok := <-c; !ok {
						fmt.Println("Oh no, an error!")
						gm.stop()
					}
				}()
			}
		}()
	}

	<-done
}
