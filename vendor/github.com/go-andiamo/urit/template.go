package urit

import (
	"errors"
	"github.com/go-andiamo/splitter"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type PathVarsType int

const (
	Positions PathVarsType = iota
	Names
)

// NewTemplate creates a new URI template from the path provided
//
// returns an error if the path cannot be parsed into a template
//
// The options can be any FixedMatchOption or VarMatchOption - which can be used
// to extend or check fixed or variable path parts
func NewTemplate(path string, options ...interface{}) (Template, error) {
	fs, vs, so := separateParseOptions(options)
	return (&template{
		originalTemplate: slashPrefix(path),
		pathParts:        make([]pathPart, 0),
		posVarsCount:     0,
		fixedMatchOpts:   fs,
		varMatchOpts:     vs,
		pathSplitOpts:    so,
	}).parse()
}

// MustCreateTemplate is the same as NewTemplate, except that it panics on error
func MustCreateTemplate(path string, options ...interface{}) Template {
	if t, err := NewTemplate(path, options...); err != nil {
		panic(err)
	} else {
		return t
	}
}

// Template is the interface for a URI template
type Template interface {
	// PathFrom generates a path from the template given the specified path vars
	PathFrom(vars PathVars, options ...interface{}) (string, error)
	// RequestFrom generates a http.Request from the template given the specified path vars
	RequestFrom(method string, vars PathVars, body io.Reader, options ...interface{}) (*http.Request, error)
	// Matches checks whether the specified path matches the template -
	// and if a successful match, returns the extracted path vars
	Matches(path string, options ...interface{}) (PathVars, bool)
	// MatchesUrl checks whether the specified URL path matches the template -
	// and if successful match, returns the extracted path vars
	MatchesUrl(u url.URL, options ...interface{}) (PathVars, bool)
	// MatchesRequest checks whether the specified request matches the template -
	// and if a successful match, returns the extracted path vars
	MatchesRequest(req *http.Request, options ...interface{}) (PathVars, bool)
	// Sub generates a new template with added sub-path
	Sub(path string, options ...interface{}) (Template, error)
	// ResolveTo generates a new template, filling in any known path vars from the supplied vars
	ResolveTo(vars PathVars) (Template, error)
	// VarsType returns the path vars type (Positions or Names)
	VarsType() PathVarsType
	// Vars returns the path vars of the template
	Vars() []PathVar
	// OriginalTemplate returns the original (or generated) path template string
	OriginalTemplate() string
	// Template returns the template (optionally with any path var patterns removed)
	Template(removePatterns bool) string
}

type template struct {
	originalTemplate string
	pathParts        []pathPart
	posVarsCount     int
	nameVarsCount    int
	varsType         PathVarsType
	fixedMatchOpts   fixedMatchOptions
	varMatchOpts     varMatchOptions
	pathSplitOpts    []splitter.Option
}

// PathFrom generates a path from the template given the specified path vars
func (t *template) PathFrom(vars PathVars, options ...interface{}) (string, error) {
	hostOption, queryOption, _, varMatches := separatePathOptions(options)
	return t.buildPath(vars, hostOption, queryOption, varMatches)
}

func (t *template) buildPath(vars PathVars, hostOption HostOption, queryOption QueryParamsOption, varMatches varMatchOptions) (string, error) {
	var pb strings.Builder
	if hostOption != nil {
		pb.WriteString(hostOption.GetAddress())
	}
	tracker := &positionsTracker{
		vars:           vars,
		varPosition:    0,
		pathPosition:   0,
		namedPositions: map[string]int{},
		varMatches:     varMatches,
	}
	for _, pt := range t.pathParts {
		if str, err := pt.pathFrom(tracker); err == nil {
			pb.WriteString(str)
		} else {
			return "", err
		}
		tracker.pathPosition++
	}
	if queryOption != nil {
		if q, err := queryOption.GetQuery(); err == nil {
			pb.WriteString(q)
		} else {
			return "", err
		}
	}
	return pb.String(), nil
}

// RequestFrom generates a http.Request from the template given the specified path vars
func (t *template) RequestFrom(method string, vars PathVars, body io.Reader, options ...interface{}) (*http.Request, error) {
	hostOption, queryOption, headerOption, varMatches := separatePathOptions(options)
	url, err := t.buildPath(vars, hostOption, queryOption, varMatches)
	if err != nil {
		return nil, err
	}
	result, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if headerOption != nil {
		hds, err := headerOption.GetHeaders()
		if err != nil {
			return nil, err
		}
		for k, v := range hds {
			result.Header.Set(k, v)
		}
	}
	return result, nil
}

// Matches checks whether the specified path matches the template -
// and if a successful match, returns the extracted path vars
func (t *template) Matches(path string, options ...interface{}) (PathVars, bool) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, false
	}
	return t.matches(u.Path, options...)
}

// MatchesUrl checks whether the specified URL path matches the template -
// and if successful match, returns the extracted path vars
func (t *template) MatchesUrl(u url.URL, options ...interface{}) (PathVars, bool) {
	return t.matches(u.Path, options...)
}

// MatchesRequest checks whether the specified request matches the template -
// and if a successful match, returns the extracted path vars
func (t *template) MatchesRequest(req *http.Request, options ...interface{}) (PathVars, bool) {
	return t.matches(req.URL.Path, options...)
}

func (t *template) matches(path string, options ...interface{}) (PathVars, bool) {
	pts, err := matchPathSplitter.Split(path)
	if err != nil || len(pts) != len(t.pathParts) {
		return nil, false
	}
	result := newPathVars(t.varsType)
	fixedOpts, varOpts := t.mergeParseOptions(options)
	ok := true
	for i, pt := range t.pathParts {
		ok = pt.match(pts[i], i, result, fixedOpts, varOpts)
		if !ok {
			break
		}
	}
	return result, ok
}

// Sub generates a new template with added sub-path
func (t *template) Sub(path string, options ...interface{}) (Template, error) {
	add, err := NewTemplate(path, options...)
	if err != nil {
		return nil, err
	}
	ra, _ := add.(*template)
	if (ra.posVarsCount > 0 && t.nameVarsCount > 0) || (t.posVarsCount > 0 && ra.nameVarsCount > 0) {
		return nil, newTemplateParseError("template cannot contain both positional and named path variables", 0, nil)
	}
	result := t.clone()
	if strings.HasSuffix(result.originalTemplate, "/") {
		result.originalTemplate = result.originalTemplate[:len(result.originalTemplate)-1] + ra.originalTemplate
	} else {
		result.originalTemplate = result.originalTemplate + ra.originalTemplate
	}
	for _, pt := range ra.pathParts {
		result.pathParts = append(result.pathParts, pt)
	}
	result.posVarsCount += ra.posVarsCount
	result.nameVarsCount += ra.nameVarsCount
	return result, nil
}

// ResolveTo generates a new template, filling in any known path vars from the supplied vars
func (t *template) ResolveTo(vars PathVars) (Template, error) {
	tracker := &positionsTracker{
		vars:           vars,
		varPosition:    0,
		namedPositions: map[string]int{},
	}
	result := &template{
		pathParts:     make([]pathPart, 0, len(t.pathParts)),
		posVarsCount:  0,
		nameVarsCount: 0,
	}
	var orgBuilder strings.Builder
	for _, pt := range t.pathParts {
		if pt.fixed {
			orgBuilder.WriteString(`/` + pt.fixedValue)
			result.pathParts = append(result.pathParts, pt)
		} else if len(pt.subParts) == 0 {
			if str, err := tracker.getVar(&pt); err == nil {
				orgBuilder.WriteString(`/` + str)
				result.pathParts = append(result.pathParts, pathPart{
					fixed:      true,
					fixedValue: str,
				})
			} else {
				result.pathParts = append(result.pathParts, pt)
				if pt.name == "" {
					result.posVarsCount++
					orgBuilder.WriteString(`/?`)
				} else {
					orgBuilder.WriteString(`/{` + pt.name)
					result.nameVarsCount++
					if pt.orgRegexp != "" {
						orgBuilder.WriteString(`:` + pt.orgRegexp)
					}
					orgBuilder.WriteString(`}`)
				}
			}
		} else {
			np := pathPart{
				fixed:    false,
				subParts: make([]pathPart, 0, len(pt.subParts)),
			}
			resolvedCount := 0
			for _, sp := range pt.subParts {
				if sp.fixed {
					resolvedCount++
					np.subParts = append(np.subParts, sp)
				} else if str, err := tracker.getVar(&sp); err == nil {
					resolvedCount++
					np.subParts = append(np.subParts, pathPart{
						fixed:      true,
						fixedValue: str,
					})
				} else {
					np.subParts = append(np.subParts, sp)
					result.nameVarsCount++
				}
			}
			orgBuilder.WriteString(`/`)
			if resolvedCount == len(pt.subParts) {
				fxnp := pathPart{
					fixed:      true,
					fixedValue: "",
				}
				for _, sp := range np.subParts {
					orgBuilder.WriteString(sp.fixedValue)
					fxnp.fixedValue += sp.fixedValue
				}
				np = fxnp
			} else {
				for _, sp := range np.subParts {
					if sp.fixed {
						orgBuilder.WriteString(sp.fixedValue)
					} else {
						orgBuilder.WriteString(`{` + sp.name)
						if sp.orgRegexp != "" {
							orgBuilder.WriteString(`:` + sp.orgRegexp)
						}
						orgBuilder.WriteString(`}`)
					}
				}
			}
			result.pathParts = append(result.pathParts, np)
		}
	}
	result.originalTemplate = orgBuilder.String()
	return result, nil
}

// VarsType returns the path vars type (Positions or Names)
func (t *template) VarsType() PathVarsType {
	if t.posVarsCount != 0 {
		return Positions
	}
	return Names
}

// Vars returns the path vars of the template
func (t *template) Vars() []PathVar {
	result := make([]PathVar, 0, len(t.pathParts))
	namePosns := map[string]int{}
	for _, p := range t.pathParts {
		result = p.getVars(result, namePosns)
	}
	return result
}

// OriginalTemplate returns the original (or generated) path template string
func (t *template) OriginalTemplate() string {
	return t.originalTemplate
}

// Template returns the template (optionally with any path var patterns removed)
func (t *template) Template(removePatterns bool) string {
	if removePatterns {
		var builder strings.Builder
		if len(t.pathParts) == 0 {
			builder.WriteString("/")
		}
		for _, pt := range t.pathParts {
			builder.WriteString("/")
			pt.buildNoPattern(&builder)
		}
		return builder.String()
	}
	return t.originalTemplate
}

func separatePathOptions(options []interface{}) (host HostOption, params QueryParamsOption, headers HeadersOption, varMatches varMatchOptions) {
	for _, intf := range options {
		if h, ok := intf.(HostOption); ok {
			host = h
		} else if q, ok := intf.(QueryParamsOption); ok {
			params = q
		} else if hd, ok := intf.(HeadersOption); ok {
			headers = hd
		} else if v, ok := intf.(VarMatchOption); ok {
			varMatches = append(varMatches, v)
		}
	}
	return
}

func separateParseOptions(options []interface{}) (fixedMatchOptions, varMatchOptions, []splitter.Option) {
	seenFixed := map[FixedMatchOption]bool{}
	seenVar := map[VarMatchOption]bool{}
	fixeds := make(fixedMatchOptions, 0)
	vars := make(varMatchOptions, 0)
	splitOps := make([]splitter.Option, 0)
	for _, intf := range options {
		if f, ok := intf.(FixedMatchOption); ok && !seenFixed[f] {
			fixeds = append(fixeds, f)
			seenFixed[f] = true
		} else if v, ok := intf.(VarMatchOption); ok && !seenVar[v] {
			vars = append(vars, v)
			seenVar[v] = true
		} else if s, ok := intf.(splitter.Option); ok {
			splitOps = append(splitOps, s)
		}
	}
	return fixeds, vars, splitOps
}

func (t *template) mergeParseOptions(options []interface{}) (fixedMatchOptions, varMatchOptions) {
	if len(options) == 0 {
		return t.fixedMatchOpts, t.varMatchOpts
	} else if len(t.fixedMatchOpts) == 0 && len(t.varMatchOpts) == 0 {
		fs, vs, _ := separateParseOptions(options)
		return fs, vs
	}
	fixed := make(fixedMatchOptions, 0)
	seenFixed := map[FixedMatchOption]bool{}
	vars := make(varMatchOptions, 0)
	seenVars := map[VarMatchOption]bool{}
	for _, f := range t.fixedMatchOpts {
		seenFixed[f] = true
		fixed = append(fixed, f)
	}
	for _, v := range t.varMatchOpts {
		seenVars[v] = true
		vars = append(vars, v)
	}
	for _, o := range options {
		if f, ok := o.(FixedMatchOption); ok && !seenFixed[f] {
			seenFixed[f] = true
			fixed = append(fixed, f)
		} else if v, ok := o.(VarMatchOption); ok && !seenVars[v] {
			seenVars[v] = true
			vars = append(vars, v)
		}
	}
	return fixed, vars
}

var uriSplitter = splitter.MustCreateSplitter('/',
	splitter.MustMakeEscapable(splitter.Parenthesis, '\\'),
	splitter.MustMakeEscapable(splitter.CurlyBrackets, '\\'),
	splitter.MustMakeEscapable(splitter.SquareBrackets, '\\'),
	splitter.DoubleQuotesBackSlashEscaped, splitter.SingleQuotesBackSlashEscaped).
	AddDefaultOptions(splitter.IgnoreEmptyOuters, splitter.NotEmptyInnersMsg("path parts cannot be empty"))

type partCapture struct {
	template *template
}

func (c *partCapture) Apply(s string, pos int, totalLen int, captured int, skipped int, isLast bool, subParts ...splitter.SubPart) (string, bool, error) {
	if pt, err := c.template.newUriPathPart(s, pos, subParts); err != nil {
		return "", false, err
	} else {
		c.template.pathParts = append(c.template.pathParts, pt)
	}
	return s, true, nil
}

func (t *template) clone() *template {
	result := &template{
		originalTemplate: t.originalTemplate,
		pathParts:        make([]pathPart, 0, len(t.pathParts)),
		posVarsCount:     t.posVarsCount,
		nameVarsCount:    t.nameVarsCount,
		varsType:         t.varsType,
	}
	result.pathParts = append(result.pathParts, t.pathParts...)
	return result
}

func (t *template) parse() (Template, error) {
	if strings.Trim(t.originalTemplate, " ") == "" {
		return nil, newTemplateParseError("template empty", 0, nil)
	}
	splitOps := append(t.pathSplitOpts, &partCapture{template: t})
	_, err := uriSplitter.Split(t.originalTemplate, splitOps...)
	if t.posVarsCount > 0 && t.nameVarsCount > 0 {
		return nil, newTemplateParseError("template cannot contain both positional and named path variables", 0, nil)
	} else if t.nameVarsCount > 0 {
		t.varsType = Names
	}
	if err != nil {
		if terr := errors.Unwrap(err); terr != nil {
			if _, ok := terr.(TemplateParseError); ok {
				err = terr
			}
		}
	}
	return t, err
}

func (t *template) newUriPathPart(pt string, pos int, subParts []splitter.SubPart) (pathPart, error) {
	if len(subParts) == 1 && subParts[0].Type() == splitter.Fixed {
		if strings.HasPrefix(pt, "?") || strings.HasPrefix(pt, ":") {
			varPart := pathPart{
				fixed: false,
				name:  pt[1:],
			}
			t.addVar(varPart)
			return varPart, nil
		} else {
			return pathPart{
				fixed:      true,
				fixedValue: pt,
			}, nil
		}
	}
	return t.newVarPathPart(subParts)
}

func (t *template) addVar(pt pathPart) {
	if !pt.fixed {
		if pt.name != "" {
			t.nameVarsCount++
		} else {
			t.posVarsCount++
		}
	}
}

func (t *template) newVarPathPart(subParts []splitter.SubPart) (pathPart, error) {
	result := pathPart{
		fixed:    false,
		subParts: make([]pathPart, 0),
	}
	anyVarParts := false
	for _, sp := range subParts {
		if sp.Type() == splitter.Brackets && sp.StartRune() == '{' {
			anyVarParts = true
			str := sp.String()
			addPart := pathPart{
				fixed: false,
			}
			if err := addPart.setName(str[1:len(str)-1], sp.StartPos()); err != nil {
				return result, err
			}
			t.addVar(addPart)
			result.subParts = append(result.subParts, addPart)
		} else if sp.Type() == splitter.Quotes {
			result.subParts = append(result.subParts, pathPart{
				fixed:      true,
				fixedValue: sp.UnEscaped(),
			})
		} else {
			result.subParts = append(result.subParts, pathPart{
				fixed:      true,
				fixedValue: sp.String(),
			})
		}
	}
	if len(result.subParts) == 1 {
		return result.subParts[0], nil
	} else if !anyVarParts {
		var sb strings.Builder
		for _, s := range result.subParts {
			sb.WriteString(s.fixedValue)
		}
		return pathPart{
			fixed:      true,
			fixedValue: sb.String(),
		}, nil
	}
	return result, nil
}

var matchPathSplitter = splitter.MustCreateSplitter('/').
	AddDefaultOptions(splitter.IgnoreEmptyFirst, splitter.IgnoreEmptyLast, splitter.NotEmptyInners)

func slashPrefix(s string) string {
	if strings.Trim(s, " ") == "" {
		return s
	} else if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}
