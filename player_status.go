package main

import (
	"sync"
)

type playerStatus struct {
	mu    sync.Mutex
	count int
	chans map[chan bool]bool
}

func newPlayerStatus() *playerStatus {
	return &playerStatus{
		count: 0,
		chans: make(map[chan bool]bool),
	}
}

func (s *playerStatus) add(amount int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prev := s.count
	curr := prev + amount
	if curr < 0 {
		curr = 0
	}

	prevOnline := prev > 0
	currOnline := curr > 0

	if prevOnline != currOnline {
		for c := range s.chans {
			c <- currOnline
		}
	}
}

func (s *playerStatus) enter() {
	s.add(1)
}

func (s *playerStatus) exit() {
	s.add(-1)
}

func (s *playerStatus) isOnline() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count > 0
}

func (s *playerStatus) channel() chan bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := make(chan bool)
	s.chans[c] = true
	return c
}

func (s *playerStatus) detach(c chan bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chans[c]; ok {
		delete(s.chans, c)
		close(c)
	}
}

func (s *playerStatus) detachAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.chans {
		delete(s.chans, c)
		close(c)
	}
}
