// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package internal

import (
	"encoding/binary"
	"fmt"
	"io"
)

const limit = 1 << 26 // 64MB seems large enough

func ReadSlice(rd io.Reader) ([]byte, error) {
	var l int64
	if err := binary.Read(rd, binary.LittleEndian, &l); err != nil {
		return nil, err
	}
	if l > limit {
		return nil, fmt.Errorf("data size too large (%v > %v): likely the file is in the wrong format", l, limit)
	}
	data := make([]byte, l)
	if err := binary.Read(rd, binary.LittleEndian, data); err != nil {
		return nil, err
	}
	return data, nil
}

func WriteSlice(wr io.Writer, data []byte) error {
	if err := binary.Write(wr, binary.LittleEndian, int64(len(data))); err != nil {
		return err
	}
	return binary.Write(wr, binary.LittleEndian, data)
}
