package urit

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type pathPart struct {
	fixed         bool
	fixedValue    string
	subParts      []pathPart
	regexp        *regexp.Regexp
	orgRegexp     string
	allRegexp     *regexp.Regexp
	allRegexpIdxs map[int]int
	name          string
}

func (pt *pathPart) setName(name string, pos int) error {
	if cAt := strings.IndexByte(name, ':'); cAt != -1 {
		pt.name = strings.Trim(name[:cAt], " ")
		if pt.name == "" {
			return newTemplateParseError("path var name cannot be empty", pos, nil)
		}
		pt.orgRegexp = strings.Trim(name[cAt+1:], " ")
		if pt.orgRegexp != "" {
			rxBit := addRegexHeadAndTail(pt.orgRegexp)
			if rx, err := regexp.Compile(rxBit); err == nil {
				pt.regexp = rx
			} else {
				return newTemplateParseError("path var regexp problem", pos+cAt, err)
			}
		}
	} else {
		pt.name = strings.Trim(name, " ")
	}
	if pt.name == "" {
		return newTemplateParseError("path var name cannot be empty", pos, nil)
	}
	return nil
}

func (pt *pathPart) addFound(vars PathVars, val string) {
	if pt.name != "" {
		_ = vars.AddNamedValue(pt.name, val)
	} else {
		_ = vars.AddPositionalValue(val)
	}
}

func (pt *pathPart) match(s string, pathPos int, vars PathVars, fOpts fixedMatchOptions, vOpts varMatchOptions) bool {
	if pt.fixed {
		ok := pt.fixedValue == s
		if len(fOpts) > 0 {
			ok = fOpts.check(s, pt.fixedValue, pathPos, vars)
		}
		return ok
	} else if len(pt.subParts) == 0 {
		ok := pt.regexp == nil || pt.regexp.MatchString(s)
		if len(vOpts) > 0 {
			if rs, vok, applicable := vOpts.check(s, vars.Len(), pt.name, pt.regexp, pt.orgRegexp, pathPos, vars); applicable {
				s = rs
				ok = vok
			}
		}
		if ok {
			pt.addFound(vars, s)
			return true
		}
	} else {
		return pt.multiMatch(s, pathPos, vars, vOpts)
	}
	return false
}

func (pt *pathPart) multiMatch(s string, pathPos int, vars PathVars, vOpts varMatchOptions) bool {
	orx := pt.overallRegexp()
	sms := orx.FindStringSubmatch(s)
	if len(sms) > 0 {
		for i, sp := range pt.subParts {
			if !sp.fixed {
				str := sms[pt.allRegexpIdxs[i]]
				ok := true
				if len(vOpts) > 0 {
					if rs, vok, applicable := vOpts.check(str, vars.Len(), sp.name, sp.regexp, sp.orgRegexp, pathPos, vars); applicable {
						str = rs
						ok = vok
					}
				}
				if ok {
					sp.addFound(vars, str)
				}
			}
		}
		return true
	}
	return false
}

func (pt *pathPart) overallRegexp() *regexp.Regexp {
	if pt.allRegexp == nil {
		var rxb strings.Builder
		for i, sp := range pt.subParts {
			if sp.fixed {
				rxb.WriteString(`(\Q` + sp.fixedValue + `\E)`)
			} else if sp.orgRegexp != "" {
				rxb.WriteString(`(?P<vsp` + fmt.Sprintf("%d", i) + `>` + stripRegexHeadAndTail(sp.orgRegexp) + `)`)
			} else {
				rxb.WriteString(`(?P<vsp` + fmt.Sprintf("%d", i) + `>.*)`)
			}
		}
		if rx, err := regexp.Compile(addRegexHeadAndTail(rxb.String())); err == nil {
			pt.allRegexp = rx
			pt.allRegexpIdxs = map[int]int{}
			for i, nm := range rx.SubexpNames() {
				if i > 0 && nm != "" && strings.HasPrefix(nm, "vsp") {
					nmi, _ := strconv.Atoi(nm[3:])
					pt.allRegexpIdxs[nmi] = i
				}
			}
		}
	}
	return pt.allRegexp
}

func (pt *pathPart) pathFrom(tracker *positionsTracker) (string, error) {
	if pt.fixed {
		return `/` + pt.fixedValue, nil
	} else if len(pt.subParts) == 0 {
		if str, err := tracker.getVar(pt); err == nil {
			return `/` + str, nil
		} else {
			return "", err
		}
	}
	var pb strings.Builder
	for _, sp := range pt.subParts {
		if sp.fixed {
			pb.WriteString(sp.fixedValue)
		} else if str, err := tracker.getVar(&sp); err == nil {
			pb.WriteString(str)
		} else {
			return "", err
		}
	}
	return `/` + pb.String(), nil
}

func (pt *pathPart) getVars(vars []PathVar, namePosns map[string]int) []PathVar {
	if !pt.fixed && len(pt.subParts) > 0 {
		for _, sp := range pt.subParts {
			vars = sp.getVars(vars, namePosns)
		}
	} else if !pt.fixed {
		if pt.name == "" {
			vars = append(vars, PathVar{
				Position: len(vars),
			})
		} else {
			vars = append(vars, PathVar{
				Name:          pt.name,
				NamedPosition: namePosns[pt.name],
				Position:      len(vars),
			})
			namePosns[pt.name] = namePosns[pt.name] + 1
		}
	}
	return vars
}

func (pt *pathPart) buildNoPattern(builder *strings.Builder) {
	if pt.fixed {
		builder.WriteString(pt.fixedValue)
	} else if len(pt.subParts) > 0 {
		for _, sp := range pt.subParts {
			sp.buildNoPattern(builder)
		}
	} else {
		builder.WriteString("{" + pt.name + "}")
	}
}

type positionsTracker struct {
	vars           PathVars
	varPosition    int
	pathPosition   int
	namedPositions map[string]int
	varMatches     varMatchOptions
}

func (tr *positionsTracker) getVar(pt *pathPart) (string, error) {
	useVars := tr.vars
	if useVars == nil {
		useVars = Positional()
	}
	var err error
	if useVars.VarsType() == Positions {
		if str, ok := useVars.GetPositional(tr.varPosition); ok {
			tr.varPosition++
			return str, nil
		}
		return "", fmt.Errorf("no var for varPosition %d", tr.varPosition+1)
	} else {
		np := tr.namedPositions[pt.name]
		if str, ok := useVars.GetNamed(pt.name, np); ok {
			str, err = tr.checkVar(str, pt, tr.varPosition, tr.pathPosition)
			if err != nil {
				return "", err
			}
			tr.namedPositions[pt.name] = np + 1
			tr.varPosition++
			return str, nil
		} else if np == 0 {
			return "", fmt.Errorf("no var for '%s'", pt.name)
		}
		return "", fmt.Errorf("no var for '%s' (varPosition %d)", pt.name, np+1)
	}
}

func (tr *positionsTracker) checkVar(s string, pt *pathPart, pos int, pathPos int) (result string, err error) {
	result = s
	for _, ck := range tr.varMatches {
		if ck.Applicable(result, pos, pt.name, pt.regexp, pt.orgRegexp, pathPos, tr.vars) {
			if altS, ok := ck.Match(result, pos, pt.name, pt.regexp, pt.orgRegexp, pathPos, tr.vars); ok {
				result = altS
			} else {
				err = errors.New("no match path var")
			}
		}
	}
	return
}

func addRegexHeadAndTail(rx string) string {
	head := ""
	tail := ""
	if !strings.HasPrefix(rx, "^") {
		head = "^"
	}
	if !strings.HasSuffix(rx, "$") {
		tail = "$"
	}
	return head + rx + tail
}

func stripRegexHeadAndTail(rx string) string {
	if strings.HasPrefix(rx, "^") {
		rx = rx[1:]
	}
	if strings.HasSuffix(rx, "$") {
		rx = rx[:len(rx)-1]
	}
	return rx
}
