package userid

import (
	"strings"
)

// ParseWindowsSID parses a windows Security Identifier (SID).
func ParseWindowsSID(sid string) (version, auth string, sub []string) {
	next := func(idx int) (string, int) {
		cur := idx
		idx = strings.Index(sid[cur:], "-")
		if idx < 0 {
			return sid[cur:], idx
		}
		return sid[cur : cur+idx], cur + idx + 1

	}
	version, idx := next(2)
	auth, idx = next(idx)
	var sa string
	for {
		sa, idx = next(idx)
		sub = append(sub, sa)
		if idx < 0 {
			break
		}
	}
	return
}

// ParseWindowsUser returns the domain and user component of a windows
// username (domain\user).
func ParseWindowsUser(u string) (domain, user string) {
	idx := strings.LastIndex(u, `\`)
	if idx < 0 {
		user = u
		return
	}
	domain = u[:idx]
	user = u[idx+1:]
	return
}
