package urit

import (
	"errors"
	"net/url"
	"sort"
	"strings"
)

type QueryParamsOption interface {
	GetQuery() (string, error)
}

type QueryParams interface {
	QueryParamsOption
	Get(key string) (interface{}, bool)
	GetIndex(key string, index int) (interface{}, bool)
	Set(key string, value interface{}) QueryParams
	Add(key string, value interface{}) QueryParams
	Del(key string) QueryParams
	Has(key string) bool
	Sorted(on bool) QueryParams
	Clone() QueryParams
}

func NewQueryParams(namesAndValues ...interface{}) (QueryParams, error) {
	if len(namesAndValues)%2 != 0 {
		return nil, errors.New("must be a value for each name")
	}
	result := &queryParams{
		params: map[string][]interface{}{},
		sorted: true,
	}
	for i := 0; i < len(namesAndValues)-1; i += 2 {
		if k, ok := namesAndValues[i].(string); ok {
			result.params[k] = append(result.params[k], namesAndValues[i+1])
		} else {
			return nil, errors.New("name must be a string")
		}
	}
	return result, nil
}

type queryParams struct {
	params map[string][]interface{}
	sorted bool
}

func (qp *queryParams) GetQuery() (string, error) {
	var qb strings.Builder
	if len(qp.params) > 0 {
		names := make([]string, 0, len(qp.params))
		for name := range qp.params {
			names = append(names, name)
		}
		if qp.sorted {
			sort.Strings(names)
		}
		for _, name := range names {
			if v := qp.params[name]; len(v) == 0 || (len(v) == 1 && v[0] == nil) {
				qb.WriteString(ampersandOrQuestionMark(qb.Len() == 0))
				qb.WriteString(url.QueryEscape(name))
			} else {
				for _, qv := range v {
					qb.WriteString(ampersandOrQuestionMark(qb.Len() == 0))
					qb.WriteString(url.QueryEscape(name))
					if qv != nil {
						if str, err := getValue(qv); err == nil {
							qb.WriteString("=")
							qb.WriteString(url.QueryEscape(str))
						} else {
							return "", err
						}
					}
				}
			}
		}
	}
	return qb.String(), nil
}

func (qp *queryParams) Get(key string) (interface{}, bool) {
	if vs, ok := qp.params[key]; ok && len(vs) > 0 {
		return vs[0], true
	}
	return nil, false
}

func (qp *queryParams) GetIndex(key string, index int) (interface{}, bool) {
	if vs, ok := qp.params[key]; ok && len(vs) > 0 {
		if index >= 0 && index < len(vs) {
			return vs[index], true
		} else if index < 0 && (len(vs)+index) >= 0 {
			return vs[len(vs)+index], true
		}
	}
	return nil, false
}

func (qp *queryParams) Set(key string, value interface{}) QueryParams {
	qp.params[key] = []interface{}{value}
	return qp
}

func (qp *queryParams) Add(key string, value interface{}) QueryParams {
	qp.params[key] = append(qp.params[key], value)
	return qp
}

func (qp *queryParams) Del(key string) QueryParams {
	delete(qp.params, key)
	return qp
}

func (qp *queryParams) Has(key string) bool {
	_, ok := qp.params[key]
	return ok
}

func (qp *queryParams) Sorted(on bool) QueryParams {
	qp.sorted = on
	return qp
}

func (qp *queryParams) Clone() QueryParams {
	result := &queryParams{
		params: map[string][]interface{}{},
		sorted: qp.sorted,
	}
	for k, v := range qp.params {
		result.params[k] = append(v)
	}
	return result
}

func ampersandOrQuestionMark(first bool) string {
	if first {
		return "?"
	}
	return "&"
}
