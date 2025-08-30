//go:build !windows

package observability

import "golang.org/x/sys/unix"

func diskFreeBytes(path string) (total, free uint64, err error) {
	var st unix.Statfs_t
	if err = unix.Statfs(path, &st); err != nil {
		return 0, 0, err
	}

	total = uint64(st.Blocks) * uint64(st.Bsize)
	free = uint64(st.Bavail) * uint64(st.Bsize)

	return total, free, nil
}
