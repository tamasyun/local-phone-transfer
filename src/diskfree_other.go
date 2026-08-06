//go:build !windows

package main

func diskFreeBytes(path string) (uint64, error) {
	return ^uint64(0), nil
}
