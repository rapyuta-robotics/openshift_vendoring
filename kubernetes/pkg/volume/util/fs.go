// +build linux darwin

/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/openshift/kubernetes/pkg/api/resource"
)

// FSInfo linux returns (available bytes, byte capacity, byte usage, total inodes, inodes free, inode usage, error)
// for the filesystem that path resides upon.
func FsInfo(path string) (int64, int64, int64, int64, int64, int64, error) {
	statfs := &syscall.Statfs_t{}
	err := syscall.Statfs(path, statfs)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}

	// Available is blocks available * fragment size
	available := int64(statfs.Bavail) * int64(statfs.Bsize)

	// Capacity is total block count * fragment size
	capacity := int64(statfs.Blocks) * int64(statfs.Bsize)

	// Usage is block being used * fragment size (aka block size).
	usage := (int64(statfs.Blocks) - int64(statfs.Bfree)) * int64(statfs.Bsize)

	inodes := int64(statfs.Files)
	inodesFree := int64(statfs.Ffree)
	inodesUsed := inodes - inodesFree

	return available, capacity, usage, inodes, inodesFree, inodesUsed, nil
}

func Du(path string) (*resource.Quantity, error) {
	// Uses the same niceness level as cadvisor.fs does when running du
	// Uses -B 1 to always scale to a blocksize of 1 byte
	out, err := exec.Command("nice", "-n", "19", "du", "-s", "-B", "1", path).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed command 'du' ($ nice -n 19 du -s -B 1) on path %s with error %v", path, err)
	}
	used, err := resource.ParseQuantity(strings.Fields(string(out))[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse 'du' output %s due to error %v", out, err)
	}
	used.Format = resource.BinarySI
	return &used, nil
}

// Find uses the command `find <path> -dev -printf '.' | wc -c` to count files and directories.
// While this is not an exact measure of inodes used, it is a very good approximation.
func Find(path string) (int64, error) {
	var stdout, stdwcerr, stdfinderr bytes.Buffer
	var err error
	findCmd := exec.Command("find", path, "-xdev", "-printf", ".")
	wcCmd := exec.Command("wc", "-c")
	if wcCmd.Stdin, err = findCmd.StdoutPipe(); err != nil {
		return 0, fmt.Errorf("failed to setup stdout for cmd %v - %v", findCmd.Args, err)
	}
	wcCmd.Stdout, wcCmd.Stderr, findCmd.Stderr = &stdout, &stdwcerr, &stdfinderr
	if err = findCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to exec cmd %v - %v; stderr: %v", findCmd.Args, err, stdfinderr.String())
	}

	if err = wcCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to exec cmd %v - %v; stderr %v", wcCmd.Args, err, stdwcerr.String())
	}
	err = findCmd.Wait()
	if err != nil {
		return 0, fmt.Errorf("cmd %v failed. stderr: %s; err: %v", findCmd.Args, stdfinderr.String(), err)
	}
	err = wcCmd.Wait()
	if err != nil {
		return 0, fmt.Errorf("cmd %v failed. stderr: %s; err: %v", wcCmd.Args, stdwcerr.String(), err)
	}
	inodeUsage, err := strconv.ParseInt(strings.TrimSpace(stdout.String()), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse cmds: %v, %v output %s - %s", findCmd.Args, wcCmd.Args, stdout.String(), err)
	}
	return inodeUsage, nil
}
