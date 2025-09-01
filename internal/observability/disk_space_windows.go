//go:build windows

package observability

import "golang.org/x/sys/windows"

func diskFreeBytes(path string) (total, free uint64, err error) {
	var (
		freeBytesAvailable     uint64
		totalNumberOfBytes     uint64
		totalNumberOfFreeBytes uint64
	)
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	err = windows.GetDiskFreeSpaceEx(p, &freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	return totalNumberOfBytes, totalNumberOfFreeBytes, err
}
