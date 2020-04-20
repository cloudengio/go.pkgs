package linewrap

import (
	"bufio"
	"bytes"
	"strings"
)

// SimpleWrap wraps text to the specified width but indented
// to indent.
func SimpleWrap(indent, width int, text string) string {
	words := bufio.NewScanner(bytes.NewBufferString(text))
	words.Split(bufio.ScanWords)
	pad := strings.Repeat(" ", indent)
	offset := indent
	out := &strings.Builder{}
	newline := true
	for words.Scan() {
		word := words.Text()
		displayWidth := 1
		for _ = range word {
			displayWidth++
		}
		// Very simple 'jagginess' prevention, don't break the line
		// until doing so is worse than not doing so.
		remaining := width - offset
		overage := offset + displayWidth - width
		if (offset+displayWidth > width) && (overage > remaining) {
			out.WriteString("\n")
			offset = indent
			newline = true
		} else {
			offset += displayWidth
		}
		if newline {
			out.WriteString(pad)
			newline = false
		} else {
			out.WriteRune(' ')
		}
		out.WriteString(word)
	}
	return out.String()
}
