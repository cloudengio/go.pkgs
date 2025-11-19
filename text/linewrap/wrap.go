// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package linewrap provides basic support for wrapping text to a given width.
package linewrap

import (
	"bufio"
	"bytes"
	"strings"
	"unicode"
)

// Paragraph wraps the supplied text as a 'paragraph' with separate indentation
// for the initial and subsequent lines to the specified width.
func Paragraph(initial, indent, width int, text string) string {
	return prefixedParagraph(initial, indent, width, "", text)
}

// Block wraps the supplied text to width indented by indent spaces.
func Block(indent, width int, text string) string {
	return prefixedParagraph(indent, indent, width, "", text)
}

// Comment wraps the supplied text to width indented by indent spaces with
// each line starting with the supplied comment string. It is intended for
// formatting source code comments.
func Comment(indent, width int, comment, text string) string {
	return prefixedParagraph(indent, indent, width, comment, text)
}

func prefixedParagraph(initial, indent, width int, prefix, text string) string {
	initialPad := strings.Repeat(" ", initial) + prefix
	pad := strings.Repeat(" ", indent) + prefix
	blankPad := strings.TrimRightFunc(pad, unicode.IsSpace)
	out := &strings.Builder{}
	out.WriteString(initialPad)
	offset := len(pad)
	lines := bufio.NewScanner(bytes.NewBufferString(text))
	nBlankLines := 0
	lastWordWithPeriod := ""
	for lines.Scan() {
		line := lines.Text()

		if lastWordWithPeriod != "" {
			out.WriteRune(' ')
		}

		// Find the last word in the line to ensure that a space
		// is added after it if the line is not wrapped. Otherwise
		// the following:
		// first word
		// second word
		// would be wrapped as:
		// first wordsecond word
		words := bufio.NewScanner(bytes.NewBufferString(line))
		words.Split(bufio.ScanWords)
		for words.Scan() {
			lastWordWithPeriod = words.Text()
		}

		words = bufio.NewScanner(bytes.NewBufferString(line))
		words.Split(bufio.ScanWords)
		blankLine := true
		newLine := true
		for words.Scan() {
			word := words.Text()
			blankLine = false
			displayWidth := 1
			for range word {
				displayWidth++
			}
			// Very simple 'jagginess' prevention, don't break the line
			// until doing so is worse than not doing so.
			remaining := width - offset
			overage := offset + displayWidth - width
			if (offset+displayWidth > width) && (overage > remaining) {
				out.WriteString("\n")
				out.WriteString(pad)
				offset = len(pad)
				newLine = true
				lastWordWithPeriod = ""
			}
			if !newLine {
				out.WriteRune(' ')
			}
			newLine = false
			offset += displayWidth
			out.WriteString(word)
		}

		if blankLine {
			nBlankLines++
			if nBlankLines == 1 {
				out.WriteString("\n")
				if len(prefix) > 0 {
					out.WriteString(blankPad)
				}
				out.WriteString("\n")
				out.WriteString(pad)
				offset = len(pad)
			}
		} else {
			nBlankLines = 0
		}

	}
	return strings.TrimRight(out.String(), " ")
}

// Verbatim returns the supplied text with each nonempty
// line prefixed by indent spaces.
func Verbatim(indent int, text string) string {
	return Prefix(indent, "", text)
}

// Prefix returns the supplied text with each nonempty
// line prefixed by indent spaces and the supplied prefix.
func Prefix(indent int, prefix, text string) string {
	pad := strings.Repeat(" ", indent) + prefix
	out := &strings.Builder{}
	lines := bufio.NewScanner(bytes.NewBufferString(text))
	for lines.Scan() {
		if len(lines.Text()) > 0 {
			out.WriteString(pad)
			out.WriteString(lines.Text())
		}
		out.WriteString("\n")
	}
	return out.String()
}
