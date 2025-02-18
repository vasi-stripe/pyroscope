//go:build rbspy
// +build rbspy

// Package rbspy is a wrapper around this library called rbspy written in Rust
package rbspy

// #cgo darwin LDFLAGS: -L../../../third_party/rustdeps/target/release -lrustdeps
// #cgo linux,!musl,amd64 LDFLAGS: -L../../../third_party/rustdeps/target/release -lrustdeps -ldl -lunwind -lunwind-ptrace -lunwind-x86_64 -lrt -lm
// #cgo linux,!musl,arm64 LDFLAGS: -L../../../third_party/rustdeps/target/release -lrustdeps -ldl -lunwind -lunwind-ptrace -lunwind-aarch64 -lrt -lm
// #cgo linux,musl LDFLAGS: -L../../../third_party/rustdeps/target/release -lrustdeps
// #include "../../../third_party/rustdeps/rbspy.h"
import "C"

import (
	"errors"
	"time"
	"unsafe"

	"github.com/pyroscope-io/pyroscope/pkg/agent/spy"
)

// TODO: make this configurable
// TODO: pass lower level structures between go and rust?
var bufferLength = 1024 * 64

// TODO: we should probably find a better way of setting this
var Blocking bool

type RbSpy struct {
	dataBuf []byte
	dataPtr unsafe.Pointer

	errorBuf []byte
	errorPtr unsafe.Pointer

	pid int
}

func Start(params spy.InitParams) (spy.Spy, error) {
	dataBuf := make([]byte, bufferLength)
	dataPtr := unsafe.Pointer(&dataBuf[0])

	errorBuf := make([]byte, bufferLength)
	errorPtr := unsafe.Pointer(&errorBuf[0])

	// Sometimes rbspy doesn't initialize properly right after the process starts so we need to give it some time
	// TODO: handle this better
	time.Sleep(1 * time.Second)

	blocking := 0
	if Blocking {
		blocking = 1
	}
	r := C.rbspy_init(C.int(params.Pid), C.int(blocking), errorPtr, C.int(bufferLength))

	if r < 0 {
		return nil, errors.New(string(errorBuf[:-r]))
	}

	return &RbSpy{
		dataPtr:  dataPtr,
		dataBuf:  dataBuf,
		errorBuf: errorBuf,
		errorPtr: errorPtr,
		pid:      params.Pid,
	}, nil
}

func (s *RbSpy) Stop() error {
	r := C.rbspy_cleanup(C.int(s.pid), s.errorPtr, C.int(bufferLength))
	if r < 0 {
		return errors.New(string(s.errorBuf[:-r]))
	}
	return nil
}

// Snapshot calls callback function with stack-trace or error.
func (s *RbSpy) Snapshot(cb func(*spy.Labels, []byte, uint64) error) error {
	r := C.rbspy_snapshot(C.int(s.pid), s.dataPtr, C.int(bufferLength), s.errorPtr, C.int(bufferLength))
	if r < 0 {
		return errors.New(string(s.errorBuf[:-r]))
	}
	return cb(nil, s.dataBuf[:r], 1)
}

func init() {
	spy.RegisterSpy("rbspy", Start)
}
