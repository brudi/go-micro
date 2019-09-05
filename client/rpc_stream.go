package client

import (
	"context"
	"io"
	"sync"

	"github.com/micro/go-micro/codec"
)

// Implements the streamer interface
type rpcStream struct {
	sync.RWMutex
	id       string
	closed   chan bool
	err      error
	request  Request
	response Response
	codec    codec.Codec
	context  context.Context

	// signal whether we should send EOS
	sendEOS bool

	// release releases the connection back to the pool
	release func(err error)
}

func (r *rpcStream) isClosed() bool {
	r.RLock()
	defer r.RUnlock()

	select {
	case <-r.closed:
		return true
	default:
		return false
	}
}

func (r *rpcStream) Context() context.Context {
	return r.context
}

func (r *rpcStream) Request() Request {
	return r.request
}

func (r *rpcStream) Response() Response {
	return r.response
}

func (r *rpcStream) Send(msg interface{}) error {
	if r.isClosed() {
		r.setError(errShutdown)
		return errShutdown
	}

	r.RLock()
	req := codec.Message{
		Id:       r.id,
		Target:   r.request.Service(),
		Method:   r.request.Method(),
		Endpoint: r.request.Endpoint(),
		Type:     codec.Request,
	}
	r.RUnlock()

	if err := r.codec.Write(&req, msg); err != nil {
		r.setError(err)
		return err
	}

	return nil
}

func (r *rpcStream) Recv(msg interface{}) error {
	if r.isClosed() {
		r.setError(errShutdown)
		return errShutdown
	}

	var resp codec.Message

	if err := r.codec.ReadHeader(&resp, codec.Response); err != nil {
		if err == io.EOF && !r.isClosed() {
			r.setError(io.ErrUnexpectedEOF)
			return io.ErrUnexpectedEOF
		}
		r.setError(err)
		return err
	}

	var err error // Maybe err := r.Error()

	switch {
	case len(resp.Error) > 0:
		// We've got an error response. Give this to the request;
		// any subsequent requests will get the ReadResponseBody
		// error if there is one.
		if resp.Error != lastStreamResponseError {
			err = serverError(resp.Error)
		} else {
			err = io.EOF
		}
		if cerr := r.codec.ReadBody(nil); cerr != nil {
			err = cerr
		}
	default:
		if cerr := r.codec.ReadBody(msg); cerr != nil {
			err = cerr
		}
	}

	if err != nil {
		r.setError(err)
		return err
	}

	return nil
}

func (r *rpcStream) Error() error {
	r.RLock()
	defer r.RUnlock()
	return r.err
}

func (r *rpcStream) setError(err error) {
	r.Lock()
	defer r.Unlock()
	r.err = err
}

func (r *rpcStream) Close() error {
	r.Lock()
	defer r.Unlock()
	select {
	case <-r.closed:
		return nil
	default:
		close(r.closed)

		// send the end of stream message
		if r.sendEOS {
			// no need to check for error
			_ = r.codec.Write(&codec.Message{
				Id:       r.id,
				Target:   r.request.Service(),
				Method:   r.request.Method(),
				Endpoint: r.request.Endpoint(),
				Type:     codec.Error,
				Error:    lastStreamResponseError,
			}, nil)
		}

		err := r.codec.Close()

		// release the connection
		r.release(r.Error())

		// return the codec error
		return err
	}
}
