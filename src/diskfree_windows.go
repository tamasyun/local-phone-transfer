//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func diskFreeBytes(path string) (uint64, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetDiskFreeSpaceExW")
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var freeAvailable uint64
	ret, _, callErr := proc.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&freeAvailable)),
		0,
		0,
	)
	if ret == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceExW: %v", callErr)
	}
	return freeAvailable, nil
}
