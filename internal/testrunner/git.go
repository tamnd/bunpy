package testrunner

import (
	"bytes"
	"fmt"
	"os/exec"
)

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %v: %w\n%s", args, err, errBuf.String())
	}
	return out.String(), nil
}
