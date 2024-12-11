// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Zipcode lookups using data from www.geonames.org.
package zipcode

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type DB struct {
	lookup map[string]LatLong
}

func NewDB() *DB {
	return &DB{lookup: make(map[string]LatLong)}
}

type LatLong struct {
	Lat  float64 // Estimated latitude (wgs84)
	Long float64 // Estimated longitude (wgs84)
}

type Option func(o *options)

func WithTag(tag string) Option {
	return func(o *options) {
		o.tag = tag
	}
}

type options struct {
	tag string
}

// LatLong returns the latitude and longitude for the
// specified postal code and admin code (eg. AK 99553).
// GB and CA postal codes come in two formats, either the
// short form or long form:
//
//	GB: Eng BN91, or Eng "BN91 9AA".
//	CA: AB T0A, or AB "T0A 0A0".
func (zdb *DB) LatLong(admin, postal string) (LatLong, bool) {
	ll, ok := zdb.lookup[admin+" "+postal]
	return ll, ok
}

func (zdb *DB) Load(data []byte, _ ...Option) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if len(scanner.Text()) == 0 {
			continue
		}
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 12 {
			return fmt.Errorf("invalid line, wrong number of fields: (%v != 12) %v", len(parts), scanner.Text())
		}
		latStr, longStr := parts[9], parts[10]
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return fmt.Errorf("invalid latitude: %v: %v", latStr, err)
		}
		long, err := strconv.ParseFloat(longStr, 64)
		if err != nil {
			return fmt.Errorf("invalid longtitude: %v: %v", latStr, err)
		}
		key := parts[4] + " " + parts[1]
		zdb.lookup[key] = LatLong{Lat: lat, Long: long}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read data: %v", err)
	}
	return nil
}
