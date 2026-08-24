//go:build !linux

package cli

func terminalColumns(uintptr) int { return 0 }
