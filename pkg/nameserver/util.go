package nameserver

import "strings"

func sanitizeDomainQuestion(d string) string {
	dom := strings.ToLower(d)
	firstDot := strings.Index(d, ".")
	if firstDot > 0 {
		dom = dom[0:firstDot]
	}
	return dom
}
