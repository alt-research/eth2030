//go:build !linux

package canonicalproof

import "os/exec"

func configureProcess(cmd *exec.Cmd) {}
