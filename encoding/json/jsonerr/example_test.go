// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonerr_test

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"

	"cloudeng.io/encoding/json/jsonerr"
	"cloudeng.io/encoding/json/jsonpayload"
)

// Timeout is an ordinary error type: a struct with an Error method. It needs
// no JSON methods, since its payload is encoded by the standard struct
// encoding.
type Timeout struct {
	Op      string `json:"op"`
	Seconds int    `json:"seconds"`
}

func (e *Timeout) Error() string {
	return fmt.Sprintf("%v timed out after %vs", e.Op, e.Seconds)
}

func init() {
	jsonpayload.RegisterType[Timeout]()
}

// ExampleMarshal shows an error being sent and reconstructed as its original
// concrete type, so that errors.As can be used on the far side.
func ExampleMarshal() {
	buf, err := jsonerr.Marshal(&Timeout{Op: "read", Seconds: 30})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(buf))

	received, err := jsonerr.Unmarshal(buf)
	if err != nil {
		log.Fatal(err)
	}
	var timeout *Timeout
	fmt.Println(errors.As(received, &timeout), timeout.Op, timeout.Seconds)

	// Output:
	// {"error":"read timed out after 30s","detail":{"type":"cloudeng.io/encoding/json/jsonerr_test.Timeout","payload":{"op":"read","seconds":30}}}
	// true read 30
}

// ExampleUnmarshal shows what happens to errors that cannot be reconstructed:
// one whose type is not registered by the receiver, and one that has no state
// to encode in the first place. Both keep their message.
func ExampleUnmarshal() {
	// An error with no exported state carries only its message.
	buf, err := jsonerr.Marshal(errors.New("something went wrong"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(buf))

	received, err := jsonerr.Unmarshal(buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%T %v\n", received, received)

	// Output:
	// {"error":"something went wrong"}
	// *errors.errorString something went wrong
}

// ExampleReadWriter shows an error carried as an ordinary tagged field of a
// struct that is both encoded and decoded, which is the usual case for a type
// shared by both ends. Use Writer or Reader instead when only one direction
// is needed, as in ExampleWriter and ExampleReader.
func ExampleReadWriter() {
	type Response struct {
		Result string             `json:"result"`
		Err    jsonerr.ReadWriter `json:"err"`
	}

	out := Response{
		Result: "partial",
		Err:    jsonerr.ReadWriter{Err: &Timeout{Op: "write", Seconds: 5}},
	}
	buf, err := json.Marshal(&out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(buf))

	var in Response
	if err := json.Unmarshal(buf, &in); err != nil {
		log.Fatal(err)
	}
	var timeout *Timeout
	fmt.Println(in.Result, errors.As(in.Err.Err, &timeout), in.Err.Err)

	// Output:
	// {"result":"partial","err":{"error":"write timed out after 5s","detail":{"type":"cloudeng.io/encoding/json/jsonerr_test.Timeout","payload":{"op":"write","seconds":5}}}}
	// partial true write timed out after 5s
}

// serverResponse is written by a process that only ever reports errors, so it
// needs no decoder for them.
type serverResponse struct {
	Result string         `json:"result"`
	Err    jsonerr.Writer `json:"err"`
}

// clientResponse is read by a process that only ever receives errors. It is
// wire compatible with serverResponse: Writer and Reader are the two halves
// of the same representation.
type clientResponse struct {
	Result string         `json:"result"`
	Err    jsonerr.Reader `json:"err"`
}

// ExampleWriter shows an error being encoded by a party that only writes
// them, such as a server reporting a failure. Writer supplies MarshalJSONTo,
// so an error can be an ordinary tagged field of a struct that is itself
// encoded as JSON.
func ExampleWriter() {
	out := serverResponse{
		Result: "partial",
		Err:    jsonerr.Writer{Err: &Timeout{Op: "write", Seconds: 5}},
	}
	buf, err := json.Marshal(&out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(buf))

	// An absent error is written as an empty message.
	buf, err = json.Marshal(&serverResponse{Result: "ok"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(buf))

	// Output:
	// {"result":"partial","err":{"error":"write timed out after 5s","detail":{"type":"cloudeng.io/encoding/json/jsonerr_test.Timeout","payload":{"op":"write","seconds":5}}}}
	// {"result":"ok","err":{"error":""}}
}

// ExampleReader shows the other half: a party that only reads errors decoding
// what a Writer produced, recovering the concrete error type because it has
// registered it.
func ExampleReader() {
	buf, err := json.Marshal(&serverResponse{
		Result: "partial",
		Err:    jsonerr.Writer{Err: &Timeout{Op: "write", Seconds: 5}},
	})
	if err != nil {
		log.Fatal(err)
	}

	var in clientResponse
	if err := json.Unmarshal(buf, &in); err != nil {
		log.Fatal(err)
	}
	var timeout *Timeout
	fmt.Println(in.Result, errors.As(in.Err.Err, &timeout), timeout.Op, timeout.Seconds)

	// A response carrying no error decodes to a nil error.
	buf, err = json.Marshal(&serverResponse{Result: "ok"})
	if err != nil {
		log.Fatal(err)
	}
	in = clientResponse{}
	if err := json.Unmarshal(buf, &in); err != nil {
		log.Fatal(err)
	}
	fmt.Println(in.Result, in.Err.Err == nil)

	// Output:
	// partial true write 5
	// ok true
}
