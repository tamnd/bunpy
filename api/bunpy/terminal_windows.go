//go:build windows

package bunpy

func termColumns() int { return 80 }
func termRows() int    { return 24 }
func isTerminal() bool { return false }
