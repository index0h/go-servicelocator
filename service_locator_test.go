package servicelocator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetConstructor(t *testing.T) {
	function := func(...interface{}) (interface{}, error) {
		return "A", nil
	}

	sl := New("test")
	err := sl.SetConstructor("A", function)

	assert.Nil(t, err)
	assert.NotNil(t, sl.constructors["A"])
}

func TestSetConstructor_Duplicate(t *testing.T) {
	function := func(...interface{}) (interface{}, error) {
		return "A", nil
	}

	sl := New("test")
	sl.SetConstructor("A", function)

	assert.NotNil(t, sl.SetConstructor("A", function))
}

func TestSetConstructor_DuplicateWithPanic(t *testing.T) {
	function := func(...interface{}) (interface{}, error) {
		return "A", nil
	}

	sl := New("test")
	sl.SetPanicMode(true)
	sl.SetConstructor("A", function)

	caller := func() {
		sl.SetConstructor("A", function)
	}

	assert.Panics(t, caller)
}

func TestGet(t *testing.T) {
	constructorA := func() string {
		return "A"
	}

	constructorB := func(serviceA string, dataB string) ([2]string, error) {
		return [2]string{serviceA, dataB}, nil
	}

	constructorC := func(serviceA string, serviceB [2]string, dataC string) ([3]string, error) {
		return [3]string{serviceA, (serviceB[0] + serviceB[1]), dataC}, nil
	}

	sl := New("test")
	errSet1 := sl.SetConstructor("NewA", constructorA)
	errSet2 := sl.SetConstructor("NewB", constructorB)
	errSet3 := sl.SetConstructor("NewC", constructorC)

	assert.Nil(t, errSet1)
	assert.Nil(t, errSet2)
	assert.Nil(t, errSet3)

	expectedA := "A"
	expectedB := [2]string{expectedA, "data_b"}
	expectedC := [3]string{expectedA, (expectedB[0] + expectedB[1]), "data_c"}
	actualA, errA := sl.Get("a")
	actualB, errB := sl.Get("b")
	actualC, errC := sl.Get("c")

	assert.Nil(t, errA)
	assert.Nil(t, errB)
	assert.Nil(t, errC)

	assert.Equal(t, expectedA, actualA)
	assert.Equal(t, expectedB, actualB)
	assert.Equal(t, expectedC, actualC)
}

func TestGet_Duplicate(t *testing.T) {
	constructor := func(arguments ...interface{}) (interface{}, error) {
		result := "A"

		return &result, nil
	}

	sl := New("test")
	sl.SetConstructor("NewA", constructor)

	actual1, err1 := sl.Get("a")
	actual2, err2 := sl.Get("a")

	assert.Nil(t, err1)
	assert.Nil(t, err2)

	assert.Equal(t, actual1, actual2)
}

func TestGetConfig(t *testing.T) {
	sl := New("test")

	expected := internalConfigMap{
		"a": internalConfig{Constructor: "NewA"},
		"b": internalConfig{Constructor: "NewB", Arguments: []interface{}{"%a%", "data_b"}},
		"c": internalConfig{Constructor: "NewC", Arguments: []interface{}{"%a%", "%b%", "data_c"}},
	}

	actual := sl.getConfig()

	assert.Equal(t, expected, actual)
}

func TestGetConfig_FileNotFound(t *testing.T) {
	sl := New("some_unknown_file")

	var result internalConfigMap
	caller := func() {
		result = sl.getConfig()
	}

	assert.Panics(t, caller)

	assert.Nil(t, result)
}

func TestGetConfig_WrongFileType(t *testing.T) {
	sl := New("test")
	sl.SetConfigType("wrong_type_here")

	var result internalConfigMap
	caller := func() {
		result = sl.getConfig()
	}

	assert.Panics(t, caller)

	assert.Nil(t, result)
}

func TestGetConfigForService(t *testing.T) {
	sl := New("test")

	expected := internalConfig{Constructor: "NewC", Arguments: []interface{}{"%a%", "%b%", "data_c"}}

	actual := sl.getConfigForService("c")

	assert.Equal(t, expected, *actual)
}
