package servicelocator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegisterConstructor(t *testing.T) {
	function := func(...interface{}) (interface{}, error) {
		return "A", nil
	}

	sl := New("test", "yaml")
	err := sl.RegisterConstructor("A", function)

	assert.Nil(t, err)
	assert.NotNil(t, sl.constructors["A"])
}

func TestRegisterConstructor_Duplicate(t *testing.T) {
	function := func(...interface{}) (interface{}, error) {
		return "A", nil
	}

	sl := New("test", "yaml")
	sl.RegisterConstructor("A", function)

	assert.NotNil(t, sl.RegisterConstructor("A", function))
}

func TestGet(t *testing.T) {
	constructorA := func(arguments ...interface{}) (interface{}, error) {
		return "A", nil
	}

	constructorB := func(arguments ...interface{}) (interface{}, error) {
		return arguments, nil
	}

	constructorC := func(arguments ...interface{}) (interface{}, error) {
		return arguments, nil
	}

	sl := New("test", "yaml")
	sl.RegisterConstructor("NewA", constructorA)
	sl.RegisterConstructor("NewB", constructorB)
	sl.RegisterConstructor("NewC", constructorC)

	expectedA := "A"
	expectedB := []interface{}{"A", "data_b"}
	expectedC := []interface{}{expectedA, expectedB, "data_c"}

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

	sl := New("test", "yaml")
	sl.RegisterConstructor("NewA", constructor)

	actual1, err1 := sl.Get("a")
	actual2, err2 := sl.Get("a")

	assert.Nil(t, err1)
	assert.Nil(t, err2)

	assert.Equal(t, actual1, actual2)
}

func TestGetConfig(t *testing.T) {
	sl := New("test", "yaml")

	expected := iternalConfigMap{
		"a": {Constructor: "NewA"},
		"b": {Constructor: "NewB", Arguments: []interface{}{"%a%", "data_b"}},
		"c": {Constructor: "NewC", Arguments: []interface{}{"%a%", "%b%", "data_c"}},
	}

	actual, err := sl.getConfig()

	assert.Nil(t, err)
	assert.Equal(t, expected, actual)
}

func TestGetConfig_FileNotFound(t *testing.T) {
	sl := New("some_unknown_file", "yaml")

	result, err := sl.getConfig()

	assert.NotNil(t, err)
	assert.Nil(t, result)
}

func TestGetConfig_WrongFileType(t *testing.T) {
	sl := New("test", "wrong_type_here")

	result, err := sl.getConfig()

	assert.NotNil(t, err)
	assert.Nil(t, result)
}

func TestGetConfigForService(t *testing.T) {
	sl := New("test", "yaml")

	expected := iternalConfig{Constructor: "NewC", Arguments: []interface{}{"%a%", "%b%", "data_c"}}

	actual, err := sl.getConfigForService("c")

	assert.Nil(t, err)
	assert.Equal(t, expected, *actual)
}
