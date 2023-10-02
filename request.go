package main

type request[T any, U any] struct {
	data     T
	response chan U
	err      chan error
}

func newRequest[T any, U any](data T) *request[T, U] {
	return &request[T, U]{data, make(chan U), make(chan error)}
}

func (r *request[T, U]) respond(u U) {
	r.response <- u
	close(r.response)
	close(r.err)
}

func (r *request[T, U]) error(err error) {
	r.err <- err
	close(r.response)
	close(r.err)
}
