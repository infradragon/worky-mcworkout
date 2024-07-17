package urit

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type Stringable interface {
	String() string
}

func getValueIf(v interface{}) (string, bool) {
	switch av := v.(type) {
	case string:
		return av, true
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", av), true
	case float32, float64:
		return fmt.Sprintf("%v", av), true
	case bool:
		if av {
			return "true", true
		}
		return "false", true
	case time.Time:
		return av.Format(time.RFC3339), true
	case *time.Time:
		return av.Format(time.RFC3339), true
	default:
		if str, ok := stringableValue(v); ok {
			return str, true
		}
	}
	return "", false
}

func getValue(v interface{}) (string, error) {
	if str, ok := getValueIf(v); ok {
		return str, nil
	}
	return "", errors.New("unknown value type")
}

func stringableValue(v interface{}) (string, bool) {
	if v == nil {
		return "", false
	}
	if sa, ok := v.(Stringable); ok {
		return sa.String(), true
	}
	rt := reflect.TypeOf(v)
	if rt.Kind() == reflect.Func {
		if rt.NumIn() == 0 && rt.NumOut() == 1 && rt.Out(0).Kind() == reflect.String {
			rv := reflect.ValueOf(v)
			rvs := rv.Call(nil)
			return rvs[0].String(), true
		}
		return "", false
	}
	if data, err := json.Marshal(v); err == nil {
		str := string(data[:])
		if strings.HasPrefix(str, `"`) && strings.HasSuffix(str, `"`) {
			return str[1 : len(str)-1], true
		}
		return str, true
	}
	return "", false
}
