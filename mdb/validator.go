package mdb

import (
	"commons/as"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type Validator interface {
	Validate(value interface{}) (bool, error)
}

type PatternValidator struct {
	Pattern *regexp.Regexp
}

func (self *PatternValidator) Validate(obj interface{}) (bool, error) {
	value, ok := obj.(string)
	if !ok {
		return false, errors.New("syntex error")
	}

	if nil != self.Pattern {
		if !self.Pattern.MatchString(value) {
			return false, errors.New("'" + value + "' is not match '" + self.Pattern.String() + "'")
		}
	}
	return true, nil
}

type StringLengthValidator struct {
	MinLength, MaxLength int
}

func (self *StringLengthValidator) Validate(obj interface{}) (bool, error) {
	value, ok := obj.(string)
	if !ok {
		return false, errors.New("syntex error")
	}

	if 0 <= self.MinLength && self.MinLength > len(value) {
		return false, errors.New("length of '" + value + "' is less " + strconv.Itoa(self.MinLength))
	}

	if 0 <= self.MaxLength && self.MaxLength < len(value) {
		return false, errors.New("length of '" + value + "' is greate " + strconv.Itoa(self.MaxLength))
	}

	return true, nil
}

type IntegerValidator struct {
	HasMin, HasMax     bool
	MinValue, MaxValue int64
}

func (self *IntegerValidator) Validate(obj interface{}) (bool, error) {
	i64, err := as.AsInt64(obj)
	if nil != err {
		return false, errors.New("it is not a integer")
	}

	if self.HasMin && self.MinValue > i64 {
		return false, fmt.Errorf("'%d' is less minValue '%d'", i64, self.MinValue)
	}

	if self.HasMax && self.MaxValue < i64 {
		return false, fmt.Errorf("'%d' is greate maxValue '%d'", i64, self.MaxValue)
	}

	return true, nil
}

type DecimalValidator struct {
	HasMin, HasMax     bool
	MinValue, MaxValue float64
}

func (self *DecimalValidator) Validate(obj interface{}) (bool, error) {
	f64, err := as.AsFloat64(obj)
	if nil != err {
		return false, errors.New("it is not a decimal")
	}

	if self.HasMin && self.MinValue > f64 {
		return false, fmt.Errorf("'%f' is less minValue '%f'", f64, self.MinValue)
	}

	if self.HasMax && self.MaxValue < f64 {
		return false, fmt.Errorf("'%f' is greate maxValue '%f'", f64, self.MaxValue)
	}
	return true, nil
}

type DateValidator struct {
	HasMin, HasMax     bool
	MinValue, MaxValue time.Time
}

func (self *DateValidator) Validate(obj interface{}) (bool, error) {
	value, ok := obj.(time.Time)
	if !ok {
		return false, errors.New("syntex error")
	}

	if self.HasMin && self.MinValue.After(value) {
		return false, fmt.Errorf("'%s' is less minValue '%s'", value.String(), self.MinValue.String())
	}

	if self.HasMax && self.MaxValue.Before(value) {
		return false, fmt.Errorf("'%s' is greate maxValue '%s'", value.String(), self.MaxValue.String())
	}
	return true, nil
}

type EnumerationValidator struct {
	Values []interface{}
}

func (self *EnumerationValidator) Validate(obj interface{}) (bool, error) {
	var found bool = false
	for v := range self.Values {
		if v == obj {
			found = true
			break
		}
	}
	if !found {
		return false, fmt.Errorf("enum is not contains %v", obj)
	}
	return true, nil
}

type StringEnumerationValidator struct {
	Values []string
}

func (self *StringEnumerationValidator) Validate(obj interface{}) (bool, error) {
	var s string
	switch value := obj.(type) {
	case string:
		s = value
	case *string:
		s = *value
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
		return false, fmt.Errorf("enum is not contains %v", obj)
	}
	return true, nil
}
