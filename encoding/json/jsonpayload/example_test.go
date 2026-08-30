// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload_test

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"log"
	"strings"

	"cloudeng.io/encoding/json/jsonpayload"
)

// Greeting and Farewell are the message types used by the examples. A message
// type implements json.MarshalerTo to be written and json.UnmarshalerFrom to
// be read; jsonpayload.Wrapper can supply both for a type that has neither.
type Greeting struct {
	Text string `json:"text"`
}

func (g *Greeting) MarshalJSONTo(enc *jsontext.Encoder) error {
	type plain Greeting
	b, err := json.Marshal((*plain)(g))
	if err != nil {
		return err
	}
	return enc.WriteValue(jsontext.Value(b))
}

func (g *Greeting) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	v, err := dec.ReadValue()
	if err != nil {
		return err
	}
	type plain Greeting
	return json.Unmarshal(v, (*plain)(g))
}

type Farewell struct {
	Text string `json:"text"`
}

func (f *Farewell) MarshalJSONTo(enc *jsontext.Encoder) error {
	type plain Farewell
	b, err := json.Marshal((*plain)(f))
	if err != nil {
		return err
	}
	return enc.WriteValue(jsontext.Value(b))
}

func (f *Farewell) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	v, err := dec.ReadValue()
	if err != nil {
		return err
	}
	type plain Farewell
	return json.Unmarshal(v, (*plain)(f))
}

// ExampleWriter shows how a value is written as a typed message: an object
// carrying the fully qualified name of the value's type alongside its payload.
func ExampleWriter() {
	buf, err := json.Marshal(jsonpayload.NewWriter(&Greeting{Text: "hello"}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(buf))
	// Output:
	// {"type":"cloudeng.io/encoding/json/jsonpayload_test.Greeting","payload":{"text":"hello"}}
}

// ExampleDecode shows the simplest way to read a typed message whose type is
// known at compile time. Nothing needs to be registered, and the message is
// decoded directly into the caller's value.
func ExampleDecode() {
	buf, err := json.Marshal(jsonpayload.NewWriter(&Greeting{Text: "hello"}))
	if err != nil {
		log.Fatal(err)
	}

	var greeting Greeting
	if err := jsonpayload.Decode(jsontext.NewDecoder(bytes.NewReader(buf)), &greeting); err != nil {
		log.Fatal(err)
	}
	fmt.Println(greeting.Text)

	// A message for a different type is reported rather than decoded.
	var farewell Farewell
	err = jsonpayload.Decode(jsontext.NewDecoder(bytes.NewReader(buf)), &farewell)
	fmt.Println(err)

	// Output:
	// hello
	// expected type name "cloudeng.io/encoding/json/jsonpayload_test.Farewell", got "cloudeng.io/encoding/json/jsonpayload_test.Greeting"
}

// ExampleReader shows Reader adapting Decode to json.UnmarshalerFrom, so that
// a typed message can be read with json.Unmarshal. The value to decode into is
// supplied by the caller and is filled in place.
func ExampleReader() {
	buf, err := json.Marshal(jsonpayload.NewWriter(&Greeting{Text: "hello"}))
	if err != nil {
		log.Fatal(err)
	}

	var greeting Greeting
	rd := jsonpayload.NewReader(&greeting)
	if err := json.Unmarshal(buf, &rd); err != nil {
		log.Fatal(err)
	}
	fmt.Println(greeting.Text)
	// Output:
	// hello
}

// ExampleReader_nested shows the case that Reader exists for: a typed message
// carried as one field of a larger document, decoded by json/v2 itself. A type
// that implements json.UnmarshalerFrom itself can call Decode instead and do
// without the Reader.
func ExampleReader_nested() {
	type Envelope struct {
		ID      int                           `json:"id"`
		Message jsonpayload.Reader[*Greeting] `json:"message"`
	}

	inner, err := json.Marshal(jsonpayload.NewWriter(&Greeting{Text: "hello"}))
	if err != nil {
		log.Fatal(err)
	}
	buf := fmt.Appendf(nil, `{"id":7,"message":%s}`, inner)

	var greeting Greeting
	env := Envelope{Message: jsonpayload.NewReader(&greeting)}
	if err := json.Unmarshal(buf, &env); err != nil {
		log.Fatal(err)
	}
	fmt.Println(env.ID, greeting.Text)
	// Output:
	// 7 hello
}

// ExampleReaderAny shows how to read messages whose type is not known until
// the message is read. Every type that may be encountered must be registered
// so that ReaderAny can construct one.
func ExampleReaderAny() {
	jsonpayload.RegisterType[Greeting]()
	jsonpayload.RegisterType[Farewell]()

	// Each message must be written through its concrete type. Writing
	// through a variable of interface type would name the interface rather
	// than the message, since Writer takes the name of its type parameter.
	greeting, err := json.Marshal(jsonpayload.NewWriter(&Greeting{Text: "hello"}))
	if err != nil {
		log.Fatal(err)
	}
	farewell, err := json.Marshal(jsonpayload.NewWriter(&Farewell{Text: "goodbye"}))
	if err != nil {
		log.Fatal(err)
	}

	for _, buf := range [][]byte{greeting, farewell} {
		var rd jsonpayload.ReaderAny
		if err := json.Unmarshal(buf, &rd); err != nil {
			log.Fatal(err)
		}
		switch msg := rd.Value.(type) {
		case *Greeting:
			fmt.Println("greeting:", msg.Text)
		case *Farewell:
			fmt.Println("farewell:", msg.Text)
		default:
			fmt.Printf("unexpected %T\n", msg)
		}
	}

	// A message naming a type that was never registered cannot be decoded,
	// since there is nothing for ReaderAny to construct.
	var unknown jsonpayload.ReaderAny
	err = json.Unmarshal([]byte(`{"type":"example.com/pkg.Unknown","payload":{}}`), &unknown)
	fmt.Println(strings.Contains(err.Error(), `no registered type for "example.com/pkg.Unknown"`))

	// Output:
	// greeting: hello
	// farewell: goodbye
	// true
}

// ExampleWriterAny shows how to write messages that are reached through a
// variable of interface type, such as a slice of mixed message types.
// WriterAny names each value's own type, whereas Writer would name the
// interface, leaving a reader with no way to recover the message's type.
func ExampleWriterAny() {
	messages := []json.MarshalerTo{
		&Greeting{Text: "hello"},
		&Farewell{Text: "goodbye"},
	}

	var encoded [][]byte
	for _, msg := range messages {
		buf, err := json.Marshal(jsonpayload.NewWriterAny(msg))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(buf))
		encoded = append(encoded, buf)
	}

	// Each message can be read back by name, since its own type was written.
	jsonpayload.RegisterType[Greeting]()
	jsonpayload.RegisterType[Farewell]()
	for _, buf := range encoded {
		var rd jsonpayload.ReaderAny
		if err := json.Unmarshal(buf, &rd); err != nil {
			log.Fatal(err)
		}
		switch msg := rd.Value.(type) {
		case *Greeting:
			fmt.Println("greeting:", msg.Text)
		case *Farewell:
			fmt.Println("farewell:", msg.Text)
		}
	}

	// Output:
	// {"type":"cloudeng.io/encoding/json/jsonpayload_test.Greeting","payload":{"text":"hello"}}
	// {"type":"cloudeng.io/encoding/json/jsonpayload_test.Farewell","payload":{"text":"goodbye"}}
	// greeting: hello
	// farewell: goodbye
}

// ExampleReadWriter shows a typed message carried as an ordinary tagged field
// of a struct, when its type is known at compile time. Both type arguments
// must be spelled out, since Go infers type arguments for calls but not for
// types. Nothing needs to be registered and nothing is allocated: the payload
// is decoded into the field in place.
func ExampleReadWriter() {
	type Envelope struct {
		ID      int                                         `json:"id"`
		Message jsonpayload.ReadWriter[Greeting, *Greeting] `json:"message"`
	}

	out := Envelope{
		ID:      7,
		Message: jsonpayload.NewReadWriter[Greeting, *Greeting](Greeting{Text: "hello"}),
	}
	buf, err := json.Marshal(&out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(buf))

	// A zero valued Envelope can be decoded into directly.
	var in Envelope
	if err := json.Unmarshal(buf, &in); err != nil {
		log.Fatal(err)
	}
	fmt.Println(in.ID, in.Message.Value.Text)

	// Output:
	// {"id":7,"message":{"type":"cloudeng.io/encoding/json/jsonpayload_test.Greeting","payload":{"text":"hello"}}}
	// 7 hello
}

// ExampleReadWriterAny shows the same, for a field whose message type is not
// known until it is read. Each type that may be carried must be registered.
func ExampleReadWriterAny() {
	jsonpayload.RegisterType[Greeting]()
	jsonpayload.RegisterType[Farewell]()

	type Envelope struct {
		ID      int                       `json:"id"`
		Message jsonpayload.ReadWriterAny `json:"message"`
	}

	for id, msg := range []jsonpayload.ReaderWriter{&Greeting{Text: "hello"}, &Farewell{Text: "goodbye"}} {
		buf, err := json.Marshal(&Envelope{ID: id, Message: jsonpayload.NewReadWriterAny(msg)})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(buf))

		var in Envelope
		if err := json.Unmarshal(buf, &in); err != nil {
			log.Fatal(err)
		}
		switch m := in.Message.Value.(type) {
		case *Greeting:
			fmt.Println("greeting:", m.Text)
		case *Farewell:
			fmt.Println("farewell:", m.Text)
		}
	}

	// Output:
	// {"id":0,"message":{"type":"cloudeng.io/encoding/json/jsonpayload_test.Greeting","payload":{"text":"hello"}}}
	// greeting: hello
	// {"id":1,"message":{"type":"cloudeng.io/encoding/json/jsonpayload_test.Farewell","payload":{"text":"goodbye"}}}
	// farewell: goodbye
}

// ExamplePointerReaderWriter shows what a message type must provide to be
// carried by a ReadWriter. The methods must be declared on the pointer type,
// since decoding has to modify the value: Greeting, declared above, satisfies
// PointerReaderWriter[Greeting] by declaring MarshalJSONTo and
// UnmarshalJSONFrom on *Greeting. It also shows a ReadWriter used on its own
// rather than as a field of an enclosing struct.
func ExamplePointerReaderWriter() {
	// The constraint is satisfied by *Greeting rather than by Greeting,
	// which is why ReadWriter takes both as type arguments.
	var _ jsonpayload.ReaderWriter = (*Greeting)(nil)

	rw := jsonpayload.NewReadWriter(Greeting{Text: "hello"})
	buf, err := json.Marshal(rw)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(buf))

	// The zero value is ready to decode into; the payload is written into
	// the ReadWriter's own Value rather than into anything allocated for it.
	var got jsonpayload.ReadWriter[Greeting, *Greeting]
	if err := json.Unmarshal(buf, &got); err != nil {
		log.Fatal(err)
	}
	fmt.Println(got.Value.Text)

	// Output:
	// {"type":"cloudeng.io/encoding/json/jsonpayload_test.Greeting","payload":{"text":"hello"}}
	// hello
}
