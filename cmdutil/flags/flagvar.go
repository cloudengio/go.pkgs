// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package flags provides support for working with flag variables, and for
// managing flag variables by embedding them in structs. A field in a struct
// can be annotated with a tag that is used to identify it as a variable to be
// registered with a flag that contains the name of the flag, an initial
// default value and the usage message.
// This makes it convenient to colocate flags with related data structures and
// to avoid large numbers of global variables as are often encountered with
// complex, multi-level command structures.
package flags

import (
	"flag"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
	"unsafe"
)

var (
	flagValueType = reflect.TypeOf((*flag.Value)(nil)).Elem()
)

// consume up to the separator or end of data, allowing for escaping using \.
func consume(t string, sep rune) (value, remaining string) {
	val := make([]rune, 0, len(t))
	escaped := false
	for i, r := range t {
		if r == '\\' {
			escaped = true
			continue
		}
		if !escaped && r == sep {
			value = string(val)
			remaining = t[i:] // include sep
			return
		}
		escaped = false
		val = append(val, r)
	}
	value = string(val)
	remaining = ""
	return
}

func parseField(t, field string, allowEmpty, expectMore bool) (value, remaining string, err error) {
	defer func() {
		if err != nil {
			return
		}
		if !allowEmpty && len(value) == 0 {
			err = fmt.Errorf("empty field for %v", field)
			return
		}
		// are we expecting any more data after this field.
		if expectMore {
			if len(remaining) == 0 {
				err = fmt.Errorf("more fields expected after %v", field)
				return
			}
			if remaining[0] == ',' {
				remaining = remaining[1:]
			}
			return
		}
		if len(remaining) > 0 {
			err = fmt.Errorf("spurious text after %v", field)
			return
		}
	}()
	if len(t) == 0 {
		return
	}
	if t[0] == '\'' {
		value, remaining = consume(t[1:], '\'')
		if len(remaining) == 0 {
			err = fmt.Errorf("missing close quote (') for %v", field)
			return
		}
		remaining = remaining[1:]
		return
	}
	value, remaining = consume(t, ',')

	return
}

// ParseFlagTag parses the supplied string into a flag name, default literal
// value and description components. It is used by
// CreatenAndRegisterFlagsInStruct to parse the field tags.
//
// The tag format is:
//
// <name>,<default-value>,<usage>
//
// where <name> is the name of the flag, <default-value> is an optional
// literal default value for the flag and <usage> the detailed
// description for the flag.
// <default-value> may be left empty, but <name> and <usage> must
// be supplied. All fields can be quoted if they need to contain a comma.
//
// Default values may contain shell variables as per flags.ExpandEnv.
// So $HOME/.configdir may be used on both UNIX and Windows for example.
func ParseFlagTag(t string) (name, value, usage string, err error) {
	if len(t) == 0 {
		err = fmt.Errorf("empty or missing tag")
		return
	}
	name, remaining, err := parseField(t, "<name>", false, true)
	if err != nil {
		return
	}
	value, remaining, err = parseField(remaining, "<default-value>", true, true)
	if err != nil {
		return
	}
	usage, _, err = parseField(remaining, "<usage>", false, false)
	return
}

func defaultLiteralValue(typeName string) any {
	switch typeName {
	case "int":
		return int(0)
	case "int64", "time.Duration":
		return int64(0)
	case "uint":
		return uint(0)
	case "uint64":
		return uint64(0)
	case "bool":
		return bool(false)
	case "float64":
		return float64(0)
	case "string":
		return ""
	}
	return nil
}

func literalDefault(typeName, literal string, initialValue any) (value any, usageDefault string, set bool, err error) {
	if initialValue != nil {
		switch v := initialValue.(type) {
		case int, int64, uint, uint64, bool, float64, string, time.Duration:
			set = true
			value = v
			return
		}
	}
	if len(literal) == 0 {
		value = defaultLiteralValue(typeName)
		return
	}
	if tmp := ExpandEnv(literal); tmp != literal {
		usageDefault = literal
		literal = tmp
	}
	var tmp int64
	var utmp uint64
	set = true
	switch typeName {
	case "int":
		tmp, err = strconv.ParseInt(literal, 10, 64)
		if tmp > math.MaxInt {
			value = tmp
		} else {
			value = int(tmp)
		}
	case "int64":
		tmp, err = strconv.ParseInt(literal, 10, 64)
		value = tmp
	case "uint":
		utmp, err = strconv.ParseUint(literal, 10, 64)
		if utmp > math.MaxUint {
			value = utmp
		} else {
			value = uint(utmp)
		}
	case "uint64":
		utmp, err = strconv.ParseUint(literal, 10, 64)
		value = utmp
	case "bool":
		value, err = strconv.ParseBool(literal)
	case "float64":
		value, err = strconv.ParseFloat(literal, 64)
	case "time.Duration":
		value, err = time.ParseDuration(literal)
	case "string":
		value = literal
	default:
		set = false
	}
	return
}

// RegisterFlagsInStruct will selectively register fields in the supplied struct
// as flags of the appropriate type with the supplied flag.FlagSet. Fields
// are selected if they have tag of the form `cmdline:"name::<literal>,<usage>"`
// associated with them, as defined by ParseFlagTag above.
// In addition to literal default values specified in the tag it is possible
// to provide computed default values via the valuesDefaults, and also
// defaults that will appear in the usage string for help messages that
// override the actual default value. The latter is useful for flags that
// have a default that is system dependent that is not informative in the usage
// statement. For example --home-dir which should default to /home/user but the
// usage message would more usefully say --home-dir=$HOME.
// Both maps are keyed by the name of the flag, not the field.
//
// Embedded (anonymous) structs may be used provided that they are not themselves
// tagged. For example:
//
//	type CommonFlags struct {
//	  A int `cmdline:"a,,use a"`
//	  B int `cmdline:"b,,use b"`
//	}
//
//	flagSet := struct{
//	  CommonFlags
//	  C bool `cmdline:"c,,use c"`
//	}
//
// will result in three flags, --a, --b and --c.
// Note that embedding as a pointer is not supported.
func RegisterFlagsInStruct(fs *flag.FlagSet, tag string, structWithFlags any, valueDefaults map[string]any, usageDefaults map[string]string) error {
	_, err := RegisterFlagsInStructWithSetMap(fs, tag, structWithFlags, valueDefaults, usageDefaults)
	return err
}

// RegisterFlagsInStructWithSetMap is like RegisterFlagsInStruct but returns
// a SetMap which can be used to determine which flag variables have been
// initialized either with a literal in the struct tag or via the valueDefaults
// argument.
func RegisterFlagsInStructWithSetMap(fs *flag.FlagSet, tag string, structWithFlags any, valueDefaults map[string]any, usageDefaults map[string]string) (*SetMap, error) {
	reg := &registrar{
		fs:            fs,
		tag:           tag,
		valueDefaults: valueDefaults,
		usageDefaults: usageDefaults,
		sm:            &SetMap{set: map[any]string{}},
	}
	err := reg.registerFlagsInStruct(structWithFlags)
	if err != nil {
		return nil, err
	}
	for k := range valueDefaults {
		if fs.Lookup(k) == nil {
			return nil, fmt.Errorf("flag %v does not exist but specified as a value default", k)
		}
	}
	for k, v := range usageDefaults {
		if fs.Lookup(k) == nil {
			return nil, fmt.Errorf("flag %v does not exist but specified as a usage default", k)
		}
		fs.Lookup(k).DefValue = v
	}
	return reg.sm, nil
}

func createVarFlag(fs *flag.FlagSet, fieldValue reflect.Value, name, value, description string, usageDefaults map[string]string) (bool, error) {
	addr := fieldValue.Addr()
	if !addr.Type().Implements(flagValueType) {
		return false, fmt.Errorf("does not implement flag.Value")
	}
	dv := addr.Interface().(flag.Value)
	fs.Var(dv, name, description)
	set := false
	if len(value) > 0 {
		if err := dv.Set(value); err != nil {
			return false, fmt.Errorf("failed to set initial default value for flag.Value: %v", err)
		}
		set = true
	}
	if ud, ok := usageDefaults[name]; ok {
		fs.Lookup(name).DefValue = ud
	} else {
		fs.Lookup(name).DefValue = value
	}
	return set, nil
}

func createFlagsBasedOnValue(fs *flag.FlagSet, initialValue any, fieldType reflect.StructField, fieldValue reflect.Value, name, description string) bool {
	switch dv := initialValue.(type) {
	case int:
		ptr := (*int)(unsafe.Pointer(fieldValue.Addr().Pointer()))
		fs.IntVar(ptr, name, dv, description)
	case int64:
		if fieldType.Type == reflect.TypeOf(time.Duration(0)) {
			// If no default value is specified, time.Duration defaults appear
			// here as int64s rather than time.Duration with a zero value.
			ptr := (*time.Duration)(unsafe.Pointer(fieldValue.Addr().Pointer()))
			fs.DurationVar(ptr, name, 0, description)
			break
		}
		ptr := (*int64)(unsafe.Pointer(fieldValue.Addr().Pointer()))
		fs.Int64Var(ptr, name, dv, description)
	case uint:
		ptr := (*uint)(unsafe.Pointer(fieldValue.Addr().Pointer()))
		fs.UintVar(ptr, name, dv, description)
	case uint64:
		ptr := (*uint64)(unsafe.Pointer(fieldValue.Addr().Pointer()))
		fs.Uint64Var(ptr, name, dv, description)
	case bool:
		ptr := (*bool)(unsafe.Pointer(fieldValue.Addr().Pointer()))
		fs.BoolVar(ptr, name, dv, description)
	case float64:
		ptr := (*float64)(unsafe.Pointer(fieldValue.Addr().Pointer()))
		fs.Float64Var(ptr, name, dv, description)
	case string:
		ptr := (*string)(unsafe.Pointer(fieldValue.Addr().Pointer()))
		fs.StringVar(ptr, name, dv, description)
	case time.Duration:
		ptr := (*time.Duration)(unsafe.Pointer(fieldValue.Addr().Pointer()))
		fs.DurationVar(ptr, name, dv, description)
	default:
		return false
	}
	return true
}

func getTypeVal(structWithFlags any) (reflect.Type, reflect.Value, error) {
	typ := reflect.TypeOf(structWithFlags)
	val := reflect.ValueOf(structWithFlags)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = reflect.Indirect(val)
	}
	if !val.CanAddr() {
		return nil, reflect.Value{}, fmt.Errorf("%T is not addressable", structWithFlags)
	}

	if typ.Kind() != reflect.Struct {
		return nil, reflect.Value{}, fmt.Errorf("%T is not a pointer to a struct", structWithFlags)
	}
	return typ, val, nil
}

type registrar struct {
	fs            *flag.FlagSet
	tag           string
	valueDefaults map[string]any
	usageDefaults map[string]string
	sm            *SetMap
}

func (reg *registrar) possiblyEmbedded(fieldType reflect.StructField, addr reflect.Value) error {
	if fieldType.Type.Kind() == reflect.Struct && fieldType.Anonymous {
		if err := reg.registerFlagsInStruct(addr.Interface()); err != nil {
			return err
		}
	}
	return nil
}

func (reg *registrar) registerFlagsInStruct(structWithFlags any) error {
	typ, val, err := getTypeVal(structWithFlags)
	if err != nil {
		return err
	}

	for i := range typ.NumField() {
		fieldType := typ.Field(i)
		fieldValue := val.Field(i)
		fieldName := fieldType.Name
		fieldTypeName := fieldType.Type.String()

		tags, ok := fieldType.Tag.Lookup(reg.tag)
		if !ok {
			if err := reg.possiblyEmbedded(fieldType, val.Field(i).Addr()); err != nil {
				return err
			}
			continue
		}

		name, value, description, err := ParseFlagTag(tags)
		if err != nil {
			return fmt.Errorf("field %v: failed to parse tag: %v", fieldType.Name, tags)
		}
		if reg.fs.Lookup(name) != nil {
			return fmt.Errorf("flag %v already defined for this flag.FlagSet", name)
		}

		errPrefix := func() string {
			return fmt.Sprintf("field: %v of type %v for flag %v", fieldName, fieldTypeName, name)
		}

		if fieldType.Type.Kind() == reflect.Ptr {
			return fmt.Errorf("%v: field can't be a pointer", errPrefix())
		}

		initialValue, usageDefault, set, err := literalDefault(fieldTypeName, value, reg.valueDefaults[name])
		if err != nil {
			return fmt.Errorf("%v: failed to set initial default value: %v", errPrefix(), err)
		}

		if set {
			reg.sm.set[fieldValue.Addr().Pointer()] = name
		}

		if initialValue == nil {
			set, err := createVarFlag(reg.fs, fieldValue, name, value, description, reg.usageDefaults)
			if err != nil {
				return fmt.Errorf("%v: %v", errPrefix(), err)
			}
			if set {
				reg.sm.set[fieldValue.Addr().Pointer()] = name
			}
			continue
		}
		if !createFlagsBasedOnValue(reg.fs, initialValue, fieldType, fieldValue, name, description) {
			// should never reach here.
			panic(fmt.Sprintf("%v flag: field %v, flag %v: unsupported type %T", fieldTypeName, fieldName, name, initialValue))
		}
		if len(usageDefault) > 0 {
			reg.fs.Lookup(name).DefValue = usageDefault
		}
	}
	return nil
}

// SetMaps represents flag variables, indexed by their address, whose value
// has someone been set.
type SetMap struct {
	set map[any]string
}

// IsSet returns true if the supplied flag variable's value has been
// set, either via a string literal in the struct or via the valueDefaults
// argument to RegisterFlagsInStructWithSetMap.
func (sm *SetMap) IsSet(field any) (string, bool) {
	v, ok := sm.set[reflect.ValueOf(field).Pointer()]
	return v, ok
}

// WithDefault returns a value if it is not the zero value for its type, otherwise
// it returns the second value. This is useful for command line flags where
// a default value should be used if the user does not provide one.
func WithDefault[T comparable](a, b T) T {
	var zero T
	if a != zero {
		return a
	}
	return b
}
