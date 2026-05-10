//go:build windows

package storage

import "golang.org/x/sys/windows"

// DiskUsage retorna estatisticas do filesystem que contem root do jail.
func (j *Jail) DiskUsage() (Usage, error) {
	path, err := windows.UTF16PtrFromString(j.root)
	if err != nil {
		return Usage{}, err
	}
	var freeBytesAvailableToCaller uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(path, &freeBytesAvailableToCaller, &totalNumberOfBytes, &totalNumberOfFreeBytes); err != nil {
		return Usage{}, err
	}
	return Usage{
		TotalBytes: int64(totalNumberOfBytes),
		FreeBytes:  int64(freeBytesAvailableToCaller),
		UsedBytes:  int64(totalNumberOfBytes - totalNumberOfFreeBytes),
	}, nil
}
