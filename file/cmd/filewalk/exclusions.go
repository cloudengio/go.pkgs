package main

import (
	"fmt"
	"regexp"

	"cloudeng.io/errors"
)

type Exclusions struct {
	re []*regexp.Regexp
}

func NewExclusions(expr ...string) (Exclusions, error) {
	res := []*regexp.Regexp{}
	errs := errors.M{}
	for _, e := range expr {
		re, err := regexp.Compile(e)
		if err != nil {
			errs.Append(fmt.Errorf("failed to compile %v: %v", e, err))
			continue
		}
		res = append(res, re)
	}
	return Exclusions{re: res}, errs.Err()
}

func (e Exclusions) Exclude(path string) bool {
	for _, re := range e.re {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}
