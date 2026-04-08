// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"fmt"
	"testing"

	"cloudeng.io/cmdutil/cmdyaml"
	"gopkg.in/yaml.v3"
)

type pluginConfig struct {
	Type    string `yaml:"type"`
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

type appConfig struct {
	Name    string             `yaml:"name,omitempty"`
	Plugins []cmdyaml.Deferred `yaml:"plugins,omitempty"`
}

func TestDeferredValueFor(t *testing.T) {
	input := `
name: myapp
plugins:
  - type: http
    address: localhost
    port: 8080
  - type: grpc
    address: remotehost
    port: 9090
`
	var cfg appConfig
	if err := cmdyaml.ParseConfigString(input, &cfg); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if got, want := len(cfg.Plugins), 2; got != want {
		t.Fatalf("got %d plugins, want %d", got, want)
	}

	// ValueFor on a mapping node
	node, ok := cfg.Plugins[0].ValueFor("type")
	if !ok {
		t.Fatal("expected to find key 'type'")
	}
	if got, want := node.Value, "http"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	node, ok = cfg.Plugins[1].ValueFor("port")
	if !ok {
		t.Fatal("expected to find key 'port'")
	}
	if got, want := node.Value, "9090"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Missing key returns false
	_, ok = cfg.Plugins[0].ValueFor("nonexistent")
	if ok {
		t.Error("expected false for missing key")
	}
}

func TestDeferredValueForNonMapping(t *testing.T) {
	// A Deferred that is not a mapping node should return false.
	d := cmdyaml.Deferred{}
	d.Kind = yaml.ScalarNode
	d.Value = "scalar"

	_, ok := d.ValueFor("anything")
	if ok {
		t.Error("expected false for non-mapping node")
	}
}

func TestParseDeferredSuccess(t *testing.T) {
	input := `
name: myapp
plugins:
  - type: http
    address: localhost
    port: 8080
  - type: grpc
    address: remotehost
    port: 9090
`
	var cfg appConfig
	if err := cmdyaml.ParseConfigString(input, &cfg); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	p0, err := cmdyaml.ParseDeferred[pluginConfig](&cfg.Plugins[0])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := p0.Type, "http"; got != want {
		t.Errorf("Type: got %q, want %q", got, want)
	}
	if got, want := p0.Address, "localhost"; got != want {
		t.Errorf("Address: got %q, want %q", got, want)
	}
	if got, want := p0.Port, 8080; got != want {
		t.Errorf("Port: got %d, want %d", got, want)
	}

	p1, err := cmdyaml.ParseDeferred[pluginConfig](&cfg.Plugins[1])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := p1.Type, "grpc"; got != want {
		t.Errorf("Type: got %q, want %q", got, want)
	}
	if got, want := p1.Port, 9090; got != want {
		t.Errorf("Port: got %d, want %d", got, want)
	}
}

func TestParseDeferredTypeError(t *testing.T) {
	type strictConfig struct {
		Count int `yaml:"count"`
	}

	input := `
plugins:
  - count: not-a-number
`
	type wrapper struct {
		Plugins []cmdyaml.Deferred `yaml:"plugins"`
	}
	var cfg wrapper
	if err := cmdyaml.ParseConfigString(input, &cfg); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	_, err := cmdyaml.ParseDeferred[strictConfig](&cfg.Plugins[0])
	if err == nil {
		t.Fatal("expected error decoding invalid int, got nil")
	}
}

func TestDeferredRoundtrip(t *testing.T) {
	input := `plugins:
    - type: http
      address: localhost
      port: 8080
    - type: grpc
      address: remotehost
      port: 9090
`
	var cfg appConfig
	if err := cmdyaml.ParseConfigString(input, &cfg); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	want := []string{
		"type: http\naddress: localhost\nport: 8080\n",
		"type: grpc\naddress: remotehost\nport: 9090\n",
	}
	for i, plugin := range cfg.Plugins {
		out, err := yaml.Marshal((*yaml.Node)(&plugin))
		if err != nil {
			t.Fatalf("plugin %d: marshal error: %v", i, err)
		}
		if got := string(out); got != want[i] {
			t.Errorf("plugin %d: roundtrip mismatch:\ngot:  %q\nwant: %q", i, got, want[i])
		}
	}

	// Round-trip the whole config. Use a struct without extra fields so the
	// marshaled output is predictable (yaml.Marshal always emits all fields,
	// so a Name:"" would appear even when the original input omitted it).
	type pluginsOnly struct {
		Plugins []cmdyaml.Deferred `yaml:"plugins"`
	}
	var cfg2 pluginsOnly
	if err := cmdyaml.ParseConfigString(input, &cfg2); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	out, err := yaml.Marshal(cfg2)
	if err != nil {
		t.Fatalf("error marshaling config: %v", err)
	}
	if got := string(out); got != input {
		t.Errorf("roundtrip mismatch:\ngot:  %s\nwant: %s", got, input)
	}
}

func ExampleDeferred() {
	type dbConfig struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	}
	type cacheConfig struct {
		MaxSize int    `yaml:"max_size"`
		TTL     string `yaml:"ttl"`
	}
	type serviceConfig struct {
		Name     string             `yaml:"name"`
		Backends []cmdyaml.Deferred `yaml:"backends"`
	}

	input := `
name: myservice
backends:
  - host: db.example.com
    port: 5432
  - max_size: 1000
    ttl: 5m
`
	var cfg serviceConfig
	if err := cmdyaml.ParseConfigString(input, &cfg); err != nil {
		fmt.Printf("parse error: %v\n", err)
		return
	}

	db, err := cmdyaml.ParseDeferred[dbConfig](&cfg.Backends[0])
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("db host: %s, port: %d\n", db.Host, db.Port)

	cache, err := cmdyaml.ParseDeferred[cacheConfig](&cfg.Backends[1])
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("cache max_size: %d, ttl: %s\n", cache.MaxSize, cache.TTL)

	// Output:
	// db host: db.example.com, port: 5432
	// cache max_size: 1000, ttl: 5m
}

func ExampleDeferred_valueFor() {
	type routerConfig struct {
		Routes []cmdyaml.Deferred `yaml:"routes"`
	}

	input := `
routes:
  - method: GET
    path: /api/v1/users
  - method: POST
    path: /api/v1/users
`
	var cfg routerConfig
	if err := cmdyaml.ParseConfigString(input, &cfg); err != nil {
		fmt.Printf("parse error: %v\n", err)
		return
	}

	for _, route := range cfg.Routes {
		method, _ := route.ValueFor("method")
		path, _ := route.ValueFor("path")
		fmt.Printf("%s %s\n", method.Value, path.Value)
	}

	// Output:
	// GET /api/v1/users
	// POST /api/v1/users
}
