//go:build !windows

package wasmedge

import (
	"fmt"
	"os"
)

func platformDiscardWASIStdio() (
	wasiStdioDescriptors,
	wasiStdioResources,
	error,
) {
	var files [3]*os.File
	resources := wasiStdioResources{mode: wasiStdioDiscard}
	streams := [...]struct {
		name  string
		flags int
	}{
		{"stdin", os.O_RDONLY},
		{"stdout", os.O_WRONLY},
		{"stderr", os.O_WRONLY},
	}
	for i, stream := range streams {
		file, err := os.OpenFile(os.DevNull, stream.flags, 0)
		if err != nil {
			resources.release()
			return wasiStdioDescriptors{}, wasiStdioResources{}, fmt.Errorf(
				"open WASI discard %s (%v): %w",
				stream.name, err, ErrUnavailable,
			)
		}
		files[i] = file
		resources.owned = append(resources.owned, file)
	}
	stdio, err := platformWASIStdio(files)
	if err != nil {
		resources.release()
		return wasiStdioDescriptors{}, wasiStdioResources{}, err
	}
	resources.ownedDescriptors = [3]int32{
		stdio.stdin,
		stdio.stdout,
		stdio.stderr,
	}
	resources.expectedHandlers = stdio.expectedHandlers
	return stdio, resources, nil
}

func platformWASIStdio(stdio [3]*os.File) (wasiStdioDescriptors, error) {
	const maxWASIFileDescriptor = uintptr(1<<31 - 1)
	files := [...]struct {
		name string
		file *os.File
		fd   uintptr
	}{
		{"stdin", stdio[0], stdio[0].Fd()},
		{"stdout", stdio[1], stdio[1].Fd()},
		{"stderr", stdio[2], stdio[2].Fd()},
	}
	for _, file := range files {
		if file.fd > maxWASIFileDescriptor {
			return wasiStdioDescriptors{}, fmt.Errorf(
				"WASI %s file descriptor %d does not fit int32: %w",
				file.name, file.fd, ErrInvalidArgument)
		}
		if _, err := file.file.Stat(); err != nil {
			return wasiStdioDescriptors{}, fmt.Errorf(
				"WASI %s file descriptor %d is not open (%v): %w",
				file.name, file.fd, err, ErrInvalidArgument,
			)
		}
	}
	return wasiStdioDescriptors{
		custom: true,
		stdin:  int32(files[0].fd),
		stdout: int32(files[1].fd),
		stderr: int32(files[2].fd),
		expectedHandlers: [3]uint64{
			uint64(files[0].fd),
			uint64(files[1].fd),
			uint64(files[2].fd),
		},
	}, nil
}
