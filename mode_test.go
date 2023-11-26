package main

import "testing"

func TestGameMode(t *testing.T) {
	if humanMode.String() != "human" {
		t.FailNow()
	}
	if machineMode.String() != "machine" {
		t.FailNow()
	}
	if gameMode(123).String() != "invalid" {
		t.FailNow()
	}
	if mode, err := ModeFromString("human"); err != nil || mode != humanMode {
		t.FailNow()
	}
	if mode, err := ModeFromString("machine"); err != nil || mode != machineMode {
		t.FailNow()
	}
	if _, err := ModeFromString("dadsa"); err == nil {
		t.FailNow()
	}
}
