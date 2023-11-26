package main

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/luc527/go_checkers/core"
)

func generateRandomPlyHistory() []core.Ply {
	var history []core.Ply
	g := core.NewGame()
	for !g.Result().Over() {
		plies := g.Plies()
		ply := plies[rand.Intn(len(plies))]
		history = append(history, ply)
		g.DoPly(ply)
	}
	return history
}

func TestPlyHistoryStorage(t *testing.T) {
	actualHistory := generateRandomPlyHistory()

	id, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}

	db := &memStore{}

	savePlyHistory(db, machineMode, id, actualHistory)

	savedHistory, err := getPlyHistory(db, machineMode, id)
	if err != nil {
		t.Fatal(err)
	}

	if len(actualHistory) != len(savedHistory) {
		t.Fatal("length mismatch")
	}

	for i, savedPly := range savedHistory {
		if !slices.Equal(savedPly, actualHistory[i]) {
			t.Fatalf("mismatch at [%d]", i)
		}
	}
}

func TestGetStoredGame(t *testing.T) {
	db := &memStore{}

	var ids []uuid.UUID
	var modes []gameMode

	for i := 0; i < 20; i++ {
		mode := humanMode
		if rand.Float32() < 0.5 {
			mode = machineMode
		}
		id, err := uuid.NewRandom()
		if err != nil {
			t.Fatal(err)
		}
		history := generateRandomPlyHistory()
		ids = append(ids, id)
		modes = append(modes, mode)
		if err := savePlyHistory(db, mode, id, history); err != nil {
			t.Fatal(err)
		}
	}

	for i, id := range ids {
		mode := modes[i]
		plies, err := getPlyHistory(db, mode, id)
		if err != nil {
			t.Fatal(err)
		}
		if len(plies) == 0 {
			t.Fatal("not really stored")
		}
	}

	for _, m := range []gameMode{humanMode, machineMode} {
		var want []uuid.UUID
		for i, id := range ids {
			if modes[i] == m {
				want = append(want, id)
			}
		}

		slices.SortFunc(want, func(a, b uuid.UUID) int {
			return compareBytes(a[:], b[:])
		})

		got, err := getGameIds(db, m)
		if err != nil {
			t.Fatal(err)
		}

		eq := slices.EqualFunc(want, got, func(u1, u2 uuid.UUID) bool {
			cmp := compareBytes(u1[:], u2[:])
			fmt.Printf("comparing %v and %v, result: %v\n", u1, u2, cmp)
			return cmp == 0
		})
		if !eq {
			t.Fatalf("not equal (%v)", m)
		}
	}
}
