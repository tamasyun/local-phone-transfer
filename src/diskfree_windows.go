//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func freeDiskBytes(path string) (uint64, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetDiskFreeSpaceExW")
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var freeAvailable uint64
	var totalBytes uint64
	var totalFree uint64
	r1, _, callErr := proc.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&freeAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if r1 == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceExW: %w", callErr)
	}
	return freeAvailable, nil
}
