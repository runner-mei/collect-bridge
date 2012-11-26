package mdb

import (
	"regexp"
	"testing"
	"time"
)

func TestDate(t *testing.T) {

	value1, _ := time.Parse("2006-01-02 15:04:05", "2009-10-11 12:12:11")
	value2, _ := time.Parse("2006-01-02 15:04:05", "2009-10-11 12:12:12")
	value3, _ := time.Parse("2006-01-02 15:04:05", "2009-10-11 12:12:13")
	value4, _ := time.Parse("2006-01-02 15:04:05", "2009-10-11 12:12:14")
	value5, _ := time.Parse("2006-01-02 15:04:05", "2009-10-11 12:12:15")
	value6, _ := time.Parse("2006-01-02 15:04:05", "2009-10-11 12:12:16")

	var validator DateValidator

	assertTrue := func(value interface{}) {
		if ok, err := validator.Validate(value); !ok {
			t.Errorf("test Date failed, %s", err.Error())
		}
	}
	assertFalse := func(value interface{}) {
		if ok, _ := validator.Validate(value); ok {
			t.Errorf("test Date failed")
		}
	}

	assertTrue(value1)
	assertTrue(value5)
	assertTrue(value6)

	validator.HasMax = true
	validator.MaxValue = value5
	assertTrue(value1)
	assertTrue(value5)
	assertFalse(value6)

	validator.HasMax = false
	validator.MaxValue = value5
	validator.HasMin = true
	validator.MinValue = value2

	assertFalse(value1)
	assertTrue(value2)
	assertTrue(value6)

	validator.HasMax = true
	validator.MaxValue = value5
	validator.HasMin = true
	validator.MinValue = value2

	assertFalse(value1)
	assertTrue(value2)
	assertTrue(value3)
	assertTrue(value4)
	assertTrue(value5)
	assertFalse(value6)
}

func TestInteger(t *testing.T) {
	var value1 int = 1
	var value2 int = 2
	var value3 int = 3
	var value4 int = 4
	var value5 int = 5
	var value6 int = 6

	var validator IntegerValidator

	assertTrue := func(value interface{}) {
		if ok, err := validator.Validate(value); !ok {
			t.Errorf("test integer failed, %s", err.Error())
		}
	}
	assertFalse := func(value interface{}) {
		if ok, _ := validator.Validate(value); ok {
			t.Errorf("test integer failed")
		}
	}

	assertTrue(value1)
	assertTrue(value5)
	assertTrue(value6)

	validator.HasMax = true
	validator.MaxValue = int64(value5)
	assertTrue(value1)
	assertTrue(value5)
	assertFalse(value6)

	validator.HasMax = false
	validator.MaxValue = int64(value5)
	validator.HasMin = true
	validator.MinValue = int64(value2)

	assertFalse(value1)
	assertTrue(value2)
	assertTrue(value6)

	validator.HasMax = true
	validator.MaxValue = int64(value5)
	validator.HasMin = true
	validator.MinValue = int64(value2)

	assertFalse(value1)
	assertTrue(value2)
	assertTrue(value3)
	assertTrue(value4)
	assertTrue(value5)
	assertFalse(value6)
}

func TestDouble(t *testing.T) {

	var value1 float32 = 1.0
	var value2 float32 = 2.0
	var value3 float32 = 3.0
	var value4 float32 = 4.0
	var value5 float32 = 5.0
	var value6 float32 = 6.0

	var validator DecimalValidator

	assertTrue := func(value interface{}) {
		if ok, err := validator.Validate(value); !ok {
			t.Errorf("test float failed, %s", err.Error())
		}
	}
	assertFalse := func(value interface{}) {
		if ok, _ := validator.Validate(value); ok {
			t.Errorf("test float failed")
		}
	}

	assertTrue(value1)
	assertTrue(value5)
	assertTrue(value6)

	validator.HasMax = true
	validator.MaxValue = float64(value5)
	assertTrue(value1)
	assertTrue(value5)
	assertFalse(value6)

	validator.HasMax = false
	validator.MaxValue = float64(value5)
	validator.HasMin = true
	validator.MinValue = float64(value2)

	assertFalse(value1)
	assertTrue(value2)
	assertTrue(value6)

	validator.HasMax = true
	validator.MaxValue = float64(value5)
	validator.HasMin = true
	validator.MinValue = float64(value2)

	assertFalse(value1)
	assertTrue(value2)
	assertTrue(value3)
	assertTrue(value4)
	assertTrue(value5)
	assertFalse(value6)
}

func TestString(t *testing.T) {

	var value1 string = "aaa1"
	var value2 string = "aaaa2"
	var value3 string = "aaaaa3"
	var value4 string = "aaaaaa4"
	var value5 string = "aaaaaaa5"
	var value6 string = "aaaaaaaa6"

	var validator StringValidator

	assertTrue := func(value interface{}) {
		if ok, err := validator.Validate(value); !ok {
			t.Errorf("test string failed, %s", err.Error())
		}
	}
	assertFalse := func(value interface{}) {
		if ok, _ := validator.Validate(value); ok {
			t.Errorf("test string failed")
		}
	}

	validator.MaxLength = -1
	validator.MinLength = -1
	assertTrue(value1)
	assertTrue(value5)
	assertTrue(value6)

	validator.MaxLength = len(value5)
	validator.MinLength = -1
	assertTrue(value1)
	assertTrue(value5)
	assertFalse(value6)

	validator.MaxLength = -1
	validator.MinLength = len(value2)

	assertFalse(value1)
	assertTrue(value2)
	assertTrue(value6)

	validator.MaxLength = len(value5)
	validator.MinLength = len(value2)
	assertFalse(value1)
	assertTrue(value2)
	assertTrue(value3)
	assertTrue(value4)
	assertTrue(value5)
	assertFalse(value6)

	validator.MaxLength = -1
	validator.MinLength = -1

	validator.Pattern, _ = regexp.Compile("a.*")
	assertFalse("ddd")
	assertTrue("aaa")
}
