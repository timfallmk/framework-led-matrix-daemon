//go:build !windows

// Package observability provides disk space monitoring functionality for Unix-like systems.
package observability

import "golang.org/x/sys/unix"

func diskFreeBytes(path string) (total, free uint64, err error) {
	var st unix.Statfs_t
	if err = unix.Statfs(path, &st); err != nil {
		return 0, 0, err
	}

	// #nosec G115 - Unix filesystem values are safe for conversion
	total = st.Blocks * uint64(st.Bsize)
	// #nosec G115 - Unix filesystem values are safe for conversion
	free = st.Bavail * uint64(st.Bsize)

	return total, free, nil
}
