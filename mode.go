package main

import "fmt"

type gameMode byte

const (
	humanMode = gameMode(iota)
	machineMode
)

func (m gameMode) String() string {
	switch m {
	case humanMode:
		return "human"
	case machineMode:
		return "machine"
	default:
		return "invalid"
	}
}

func ModeFromString(s string) (gameMode, error) {
	switch s {
	case "human":
		return humanMode, nil
	case "machine":
		return machineMode, nil
	default:
		return 0, fmt.Errorf("invalid game mode %v", s)
	}
}
