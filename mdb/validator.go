package mdb

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type Validator interface {
	Validate(value interface{}, attributes map[string]interface{}) (bool, error)
}

type PatternValidator struct {
	Pattern *regexp.Regexp
}

func (self *PatternValidator) Validate(pv interface{}, attributes map[string]interface{}) (bool, error) {
	var s string
	switch value := pv.(type) {
	case string:
		s = value
	case *string:
		s = *value
	default:
		return false, errors.New("syntex error, it is not a string")
	}

	if nil != self.Pattern {
		if !self.Pattern.MatchString(s) {
			return false, errors.New("'" + s + "' is not match '" + self.Pattern.String() + "'")
		}
	}
	return true, nil
}

type StringLengthValidator struct {
	MinLength, MaxLength int
}

func (self *StringLengthValidator) Validate(pv interface{}, attributes map[string]interface{}) (bool, error) {
	var s string
	switch value := pv.(type) {
	case SqlString:
		s = string(value)
	case *SqlString:
		s = string(*value)
	case string:
		s = value
	case *string:
		s = *value
	default:
		return false, errors.New("syntex error, it is not a string")
	}

	if 0 <= self.MinLength && self.MinLength > len(s) {
		return false, errors.New("length of '" + s + "' is less " + strconv.Itoa(self.MinLength))
	}

	if 0 <= self.MaxLength && self.MaxLength < len(s) {
		return false, errors.New("length of '" + s + "' is greate " + strconv.Itoa(self.MaxLength))
	}

	return true, nil
}

type IntegerValidator struct {
	HasMin, HasMax     bool
	MinValue, MaxValue int64
}

func (self *IntegerValidator) Validate(pv interface{}, attributes map[string]interface{}) (bool, error) {
	i64, ok := pv.(SqlInteger64)
	if !ok {
		return false, errors.New("syntex error, it is not a integer")
	}

	if self.HasMin && self.MinValue > int64(i64) {
		return false, fmt.Errorf("'%d' is less minValue '%d'", int64(i64), self.MinValue)
	}

	if self.HasMax && self.MaxValue < int64(i64) {
		return false, fmt.Errorf("'%d' is greate maxValue '%d'", int64(i64), self.MaxValue)
	}

	return true, nil
}

type DecimalValidator struct {
	HasMin, HasMax     bool
	MinValue, MaxValue float64
}

func (self *DecimalValidator) Validate(pv interface{}, attributes map[string]interface{}) (bool, error) {
	f64, ok := pv.(SqlDecimal)
	if !ok {
		return false, errors.New("syntex error, it is not a decimal")
	}

	if self.HasMin && self.MinValue > float64(f64) {
		return false, fmt.Errorf("'%f' is less minValue '%f'", float64(f64), self.MinValue)
	}

	if self.HasMax && self.MaxValue < float64(f64) {
		return false, fmt.Errorf("'%f' is greate maxValue '%f'", float64(f64), self.MaxValue)
	}
	return true, nil
}

type DateValidator struct {
	HasMin, HasMax     bool
	MinValue, MaxValue time.Time
}

func (self *DateValidator) Validate(pv interface{}, attributes map[string]interface{}) (bool, error) {
	var t time.Time
	switch value := pv.(type) {
	case *SqlDateTime:
		t = time.Time(*value)
	case SqlDateTime:
		t = time.Time(value)
	default:
		return false, errors.New("syntex error, it is not a time")
	}

	if self.HasMin && self.MinValue.After(t) {
		return false, fmt.Errorf("'%s' is less minValue '%s'", t.String(), self.MinValue.String())
	}

	if self.HasMax && self.MaxValue.Before(t) {
		return false, fmt.Errorf("'%s' is greate maxValue '%s'", t.String(), self.MaxValue.String())
	}
	return true, nil
}

type EnumerationValidator struct {
	Values []interface{}
}

func (self *EnumerationValidator) Validate(pv interface{}, attributes map[string]interface{}) (bool, error) {
	var found bool = false
	for v := range self.Values {
		if v == pv {
			found = true
			break
		}
	}
	if !found {
		return false, fmt.Errorf("enum is not contains %v", pv)
	}
	return true, nil
}

type StringEnumerationValidator struct {
	Values []SqlString
}

func (self *StringEnumerationValidator) Validate(pv interface{}, attributes map[string]interface{}) (bool, error) {
	var s SqlString
	switch value := pv.(type) {
	case SqlString:
		s = value
	case string:
		s = SqlString(value)
	default:
		return false, errors.New("syntex error, it is not a string")
	}

	var found bool = false
	for _, v := range self.Values {
		if v == s {
			found = true
			break
		}
	}
	if !found {
		return false, fmt.Errorf("enum is not contains %v", pv)
	}
	return true, nil
}
