//go:build unix

package main

import "golang.org/x/sys/unix"

func diskFreeBytes(path string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	// Bavail = free blocks for unprivileged users
	return uint64(st.Bavail) * uint64(st.Bsize), nil
}
