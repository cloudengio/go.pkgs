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
<<<<<<< HEAD
	newLine := true
=======
>>>>>>> master
	nBlankLines := 0
	for lines.Scan() {
		words := bufio.NewScanner(bytes.NewBufferString(lines.Text()))
		words.Split(bufio.ScanWords)
		blankLine := true
<<<<<<< HEAD
=======
		newLine := true
>>>>>>> master
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
	return out.String()
}
