package filewalk

import (
	"bytes"
	"encoding/json"
	"time"
)

type Database struct{}

func (c *Contents) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Contents) UnmarshalJSON(buf []byte) error {
	dec := json.NewDecoder(buf)
	return dec.Decode(c)
}

type jsonInfo struct {
	Tag      string
	Name     string
	Size     int64
	ModTime  time.Time
	IsPrefix bool `json:,omitempty`
	IsLink   bool `json:,omitempty`
}

func (i osinfo) MarshalJSON() ([]byte, error) {
	ji := jsonInfo{
		Name:     i.Name(),
		Size:     i.Size(),
		ModTime:  i.ModTime(),
		IsPrefix: i.IsPrefix(),
		IsLink:   i.IsLink(),
	}
	return json.Marshal(&ji)
}

func (i *osinfo) UnmarshalJSON(buf []byte) error {
	ji := &jsonInfo{}
	if err := json.Unmarshal(buf, ji); err != nil {
		return err
	}
	return nil
}
