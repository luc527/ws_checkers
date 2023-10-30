package main

// TODO: show whether opponent is online in the interface
// TODO: test

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/luc527/go_checkers/core"
)

type playerConnState bool

const (
	playerOffline = playerConnState(false)
	playerOnline  = playerConnState(true)
)

type playerConnStates struct {
	mu     sync.Mutex
	states [2]playerConnState
	chans  [2]map[chan playerConnState]bool
}

func (s playerConnState) String() string {
	switch s {
	case playerOffline:
		return "offline"
	case playerOnline:
		return "online"
	}
	// unreachable
	return ""
}

func (s *playerConnStates) init() {
	s.chans[whiteColor] = make(map[chan playerConnState]bool)
	s.chans[blackColor] = make(map[chan playerConnState]bool)
}

func (s *playerConnStates) set(player core.Color, state playerConnState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[player] = state
	for c := range s.chans[player] {
		go func(c chan playerConnState) {
			c <- state
		}(c)
	}
}

func (s *playerConnStates) enter(player core.Color) {
	s.set(player, playerOnline)
}

func (s *playerConnStates) exit(player core.Color) {
	s.set(player, playerOffline)
}

func (s *playerConnStates) current(player core.Color) playerConnState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.states[player]
}

func (s *playerConnStates) channel(player core.Color) chan playerConnState {
	s.mu.Lock()
	defer s.mu.Unlock()
	l := make(chan playerConnState)
	s.chans[player][l] = true
	return l
}

func (s *playerConnStates) detach(player core.Color, l chan playerConnState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chans[player][l]; ok {
		close(l)
		delete(s.chans[player], l)
	}
}

func (s *playerConnStates) detachAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.chans[whiteColor] {
		close(c)
		delete(s.chans[whiteColor], c)
	}
	for c := range s.chans[blackColor] {
		close(c)
		delete(s.chans[blackColor], c)
	}
}

func (s playerConnState) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := buf.WriteByte('"'); err != nil {
		return nil, err
	}
	switch s {
	case playerOnline:
		buf.WriteString("online")
	case playerOffline:
		buf.WriteString("offline")
	}
	if err := buf.WriteByte('"'); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *playerConnState) UnmarshalJSON(bs []byte) error {
	if bs[0] != '"' || bs[len(bs)-1] != '"' {
		return errors.New("unmarshal playerConnState: not a string")
	}
	str := string(bs[1 : len(bs)-1])
	switch str {
	case "online":
		*s = playerOnline
	case "offline":
		*s = playerOffline
	default:
		return fmt.Errorf("unmarshal playerConnState: invalid string %q", str)
	}
	return nil
}
