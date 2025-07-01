//go:build ignore

package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"cloudeng.io/file"
	"github.com/klauspost/compress/s2"
)

type info struct {
	Path string
	Info file.Info
}

func EncodeStream(src io.Reader, dst io.Writer) error {
	enc := s2.NewWriter(dst, s2.WriterBestCompression())
	_, err := io.Copy(enc, src)
	if err != nil {
		enc.Close()
		return err
	}
	// Blocks until compression is done.
	return enc.Close()
}

func main() {
	start := time.Now()
	file, err := os.Open(os.ExpandEnv("$HOME/filewalk.gob"))
	if err != nil {
		panic(err)
	}
	in := file
	/*	in, err := gzip.NewReader(file)
		if err != nil {
			panic(err)
		}*/

	out, err := os.OpenFile(os.ExpandEnv("$HOME/filewalk.gob.s2"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	if err := EncodeStream(in, out); err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", time.Since(start))
}
