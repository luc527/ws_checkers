package main

type rawClient struct {
	incoming <-chan []byte
	outgoint chan<- []byte
	stop     <-chan struct{}
}
