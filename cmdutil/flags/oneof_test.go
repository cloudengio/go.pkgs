package flags_test

import (
	"testing"

	"cloudeng.io/cmdutil/flags"
)

func TestOneOf(t *testing.T) {
	en := "val1"
	err := flags.OneOf(en).Validate("defval", "val1", "val2")
	if err != nil {
		t.Errorf("OneOf: %v", err)
	}
	err = flags.OneOf("bad").Validate("b", "a")
	if err == nil || err.Error() != `unrecognised flag value: "bad" is not one of: a, b` {
		t.Errorf("unexpected or missing error: %v", err)
	}
}
