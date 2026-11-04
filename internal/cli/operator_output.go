package cli

// R: report output delivery failure for the additive operator inspection views.

import (
	"fmt"
	"io"
)

func writeOperatorOutput(stdout, stderr io.Writer, command string, data []byte) int {
	n, err := stdout.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return commandError(stderr, command, fmt.Errorf("write inspection output: %w", err))
	}
	return 0
}
