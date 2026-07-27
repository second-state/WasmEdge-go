//go:build windows

package wasmedge

// #include "shims.h"
import "C"

import (
	"fmt"
	"os"
	"sync"
)

func platformDiscardWASIStdio() (
	wasiStdioDescriptors,
	wasiStdioResources,
	error,
) {
	var stdin, stdout, stderr C.int32_t
	var stdinHandler, stdoutHandler, stderrHandler C.uint64_t
	status, errno := C.wasmedgego_wasiOpenNullFds(
		&stdin,
		&stdout,
		&stderr,
		&stdinHandler,
		&stdoutHandler,
		&stderrHandler,
	)
	if status != 0 {
		return wasiStdioDescriptors{}, wasiStdioResources{}, fmt.Errorf(
			"open WASI discard streams (%v): %w",
			errno, ErrUnavailable,
		)
	}
	resources := wasiStdioResources{
		mode:             wasiStdioDiscard,
		ownedDescriptors: [3]int32{int32(stdin), int32(stdout), int32(stderr)},
		expectedHandlers: [3]uint64{
			uint64(stdinHandler),
			uint64(stdoutHandler),
			uint64(stderrHandler),
		},
		releaseOwned: sync.OnceFunc(func() {
			C.wasmedgego_wasiCloseFds(stdin, stdout, stderr)
		}),
	}
	return wasiStdioDescriptors{
		custom:           true,
		stdin:            int32(stdin),
		stdout:           int32(stdout),
		stderr:           int32(stderr),
		expectedHandlers: resources.expectedHandlers,
	}, resources, nil
}

func platformWASIStdioHandlerOpen(handler uint64) bool {
	return C.wasmedgego_wasiHandlerIsOpen(C.uint64_t(handler)) != 0
}

func platformWASIStdio([3]*os.File) (wasiStdioDescriptors, error) {
	return wasiStdioDescriptors{}, fmt.Errorf(
		"custom WASI stdio is unsupported on Windows because WasmEdge expects CRT file descriptors: %w",
		ErrInvalidArgument)
}
