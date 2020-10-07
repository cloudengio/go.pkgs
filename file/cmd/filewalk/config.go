package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"cloudeng.io/cmdutil/structdoc"
	"cloudeng.io/file/diskusage"
	"gopkg.in/yaml.v2"
)

type LayoutSpec struct {
	Type   string `yaml:"type" cmd:"name of this layout"`
	Prefix string `yaml:"prefix" cmd:"prefix that this layout applies to"`
}

type Layout struct {
	LayoutSpec `yaml:",inline"`
	Config     yaml.MapSlice
}

type Config struct {
	Layouts    []Layout `yaml:"layouts" cmd:"per-prefix filesystem layouts"`
	Exclusions []string `yaml:"exclusions" cmd:"regular expressions of prefixes to exclude from the scan"`
}

func configFromFile(filename string) (*Config, error) {
	config := &Config{}
	buf, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		fmt.Printf("warning: config file %s does not exist, assuming a simple layout with 4K block size\n", filename)
		simple, _ := newSimpleLayout(&Simple{BlockSize: 4096})
		configuredLayouts = append(configuredLayouts,
			layoutInstance{prefix: "", fn: simple},
		)
		return config, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v: %v", filename, err)
	}
	err = yaml.Unmarshal(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse/process config file %v: %v", filename, err)
	}
	// sort by longest prefix first.
	sort.Slice(configuredLayouts, func(i, j int) bool {
		return len(configuredLayouts[i].prefix) > len(configuredLayouts[j].prefix)
	})
	return config, err
}

func (l *Layout) UnmarshalYAML(unmarshal func(interface{}) error) error {
	unmarshal(&l.LayoutSpec)
	cfg, ok := supportedLayouts[l.Type]
	if !ok {
		return fmt.Errorf("unsupported layout: %v %v", l.Type, l.Prefix)
	}
	if err := unmarshal(cfg.config); err != nil {
		return err
	}
	fn, err := cfg.factory(cfg.config)
	if err != nil {
		return fmt.Errorf("failed to configure %v for prefix %v: %v", l.Type, l.Prefix, err)
	}
	configuredLayouts = append(configuredLayouts,
		layoutInstance{prefix: l.Prefix, fn: fn},
	)
	return err
}

type Simple struct {
	BlockSize int64 `yaml:"block_size" cmd:"block size used by this filesystem"`
}

type Identity struct{}

type Raid0 struct {
	StripeSize int64 `yaml:"stripe_size" cmd:"the size of the raid0 stripes"`
	NumStripes int   `yaml:"num_stripes" cmd:"the number of stripes used"`
}

type layoutConfig struct {
	config  interface{}
	factory func(cfg interface{}) (diskusage.OnDiskSize, error)
}

var supportedLayouts = map[string]layoutConfig{
	"simple":   {&Simple{}, newSimpleLayout},
	"identity": {&Identity{}, newIdentity},
	"raid0":    {&Raid0{}, newRaid0},
}

type layoutInstance struct {
	prefix string
	fn     diskusage.OnDiskSize
}

var configuredLayouts []layoutInstance

func newSimpleLayout(cfg interface{}) (diskusage.OnDiskSize, error) {
	c := cfg.(*Simple)
	if s := c.BlockSize; s == 0 {
		return nil, fmt.Errorf("invalid block size: %v", s)
	}
	l := &diskusage.Simple{BlockSize: c.BlockSize}
	return l.OnDiskSize, nil
}

func newIdentity(cfg interface{}) (diskusage.OnDiskSize, error) {
	l := &diskusage.Identity{}
	return l.OnDiskSize, nil
}

func newRaid0(cfg interface{}) (diskusage.OnDiskSize, error) {
	c := cfg.(*Raid0)
	if s := c.StripeSize; s == 0 {
		return nil, fmt.Errorf("invalid stripe size: %v", s)
	}
	if s := c.NumStripes; s == 0 {
		return nil, fmt.Errorf("invalid number of stripes: %v", s)
	}
	l := &diskusage.RAID0{
		StripeSize: c.StripeSize,
		NumStripes: c.NumStripes,
	}
	return l.OnDiskSize, nil
}

func sizeFuncFor(prefix string) diskusage.OnDiskSize {
	for _, l := range configuredLayouts {
		if strings.HasPrefix(prefix, l.prefix) {
			return l.fn
		}
	}
	// default to identity.
	return func(s int64) int64 { return s }
}

func describeConfigFile() (string, error) {
	out := &strings.Builder{}
	desc, err := structdoc.Describe(&Config{}, "cmd", "YAML configuration file options\n")
	if err != nil {
		return "", err
	}
	out.WriteString(structdoc.FormatFields(0, 2, desc.Fields))
	if len(supportedLayouts) == 0 {
		return out.String(), nil
	}
	out.WriteString("\nSupported layouts\n\n")
	for t, l := range supportedLayouts {
		desc, err := structdoc.Describe(l.config, "cmd", t+":\n")
		if err != nil {
			return "", err
		}
		out.WriteString("  " + desc.Detail)
		out.WriteString(structdoc.FormatFields(4, 2, desc.Fields))
	}
	return out.String(), nil
}
