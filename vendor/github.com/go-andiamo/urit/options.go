package urit

import (
	"regexp"
	"strings"
)

// FixedMatchOption is the option interface for checking if a fixed path part matches the template
//
// An example is provided by the CaseInsensitiveFixed - which allows fixed path parts to be matched
// regardless of case
type FixedMatchOption interface {
	Match(value string, expected string, pathPos int, vars PathVars) bool
}

// VarMatchOption is the option interface for checking if a variable part matches the template
//
// It can also be used to adjust the path variable found
type VarMatchOption interface {
	Applicable(value string, position int, name string, rx *regexp.Regexp, rxs string, pathPos int, vars PathVars) bool
	Match(value string, position int, name string, rx *regexp.Regexp, rxs string, pathPos int, vars PathVars) (string, bool)
}

var (
	_CaseInsensitiveFixed = &caseInsensitiveFixed{}
	_PathRegexCheck       = &pathRegexChecker{}
)
var (
	CaseInsensitiveFixed = _CaseInsensitiveFixed // is a FixedMatchOption that can be used with templates to allow case-insensitive fixed path parts
	PathRegexCheck       = _PathRegexCheck       // is a VarMatchOption that can be used with Template.PathFrom or Template.RequestFrom to check that vars passed in match regexes for the path part
)

type fixedMatchOptions []FixedMatchOption
type varMatchOptions []VarMatchOption

func (opts fixedMatchOptions) check(value string, expected string, pathPos int, vars PathVars) bool {
	ok := false
	for _, o := range opts {
		ok = o.Match(value, expected, pathPos, vars)
		if ok {
			break
		}
	}
	return ok
}

func (opts varMatchOptions) check(value string, position int, name string, rx *regexp.Regexp, rxs string, pathPos int, vars PathVars) (string, bool, bool) {
	ok := false
	result := value
	checked := 0
	for _, o := range opts {
		if o.Applicable(value, position, name, rx, rxs, pathPos, vars) {
			checked++
			if s, oko := o.Match(value, position, name, rx, rxs, pathPos, vars); oko {
				result = s
				ok = oko
				break
			}
		}
	}
	return result, ok, checked > 0
}

type caseInsensitiveFixed struct{}

func (o *caseInsensitiveFixed) Match(value string, expected string, pathPos int, vars PathVars) bool {
	return value == expected || strings.EqualFold(value, expected)
}

type pathRegexChecker struct{}

func (o *pathRegexChecker) Applicable(value string, position int, name string, rx *regexp.Regexp, rxs string, pathPos int, vars PathVars) bool {
	return rx != nil
}

func (o *pathRegexChecker) Match(value string, position int, name string, rx *regexp.Regexp, rxs string, pathPos int, vars PathVars) (string, bool) {
	if rx != nil && !rx.MatchString(value) {
		return value, false
	}
	return value, true
}
