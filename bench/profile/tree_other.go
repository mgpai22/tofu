//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"strings"
)

const treeMethod = "not measured: the per-process tree split is written for Windows job objects only"

func observe(*exec.Cmd) (observation, error) {
	return observation{}, errors.New("bench/profile splits a process tree on Windows only")
}

func splitCommandLine(line string) []string {
	return strings.Fields(line)
}
