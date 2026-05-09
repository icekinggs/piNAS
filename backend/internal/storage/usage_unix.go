//go:build linux || darwin

package storage

import "golang.org/x/sys/unix"

// DiskUsage retorna estatísticas do filesystem que contém root do jail.
func (j *Jail) DiskUsage() (Usage, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(j.root, &st); err != nil {
		return Usage{}, err
	}
	total := int64(st.Blocks) * int64(st.Bsize)
	free := int64(st.Bavail) * int64(st.Bsize)
	return Usage{
		TotalBytes: total,
		FreeBytes:  free,
		UsedBytes:  total - free,
	}, nil
}
