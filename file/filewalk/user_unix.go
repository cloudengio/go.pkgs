// +build darwin linux

package filewalk

import (
	"strconv"
	"syscall"
)

func getUserID(sys interface{}) string {
	si, ok := sys.(*syscall.Stat_t)
	if !ok {
		return ""
	}
	return strconv.Itoa(int(si.Uid))
}
