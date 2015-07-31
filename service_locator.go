package servicelocator

import (
	"errors"
	"github.com/spf13/viper"
	"reflect"
	"regexp"
)

var dependencyRegexp *regexp.Regexp

func init() {
	dependencyRegexp = regexp.MustCompile("%(\\w+)%")
}

type iternalConfig struct {
	Arguments   []interface{}
	Constructor string
}

type iternalConfigMap map[string]iternalConfig

type ServiceLocator struct {
	panicMode bool
	configLoader *viper.Viper
	constructors map[string]reflect.Value
	services     map[string]interface{}
	config       iternalConfigMap
}

func New(fileName string, configType string) *ServiceLocator {
	configLoader := viper.New()
	configLoader.SetConfigType(configType)
	configLoader.SetConfigName(fileName)

	return &ServiceLocator{
		configLoader: configLoader,
		constructors: make(map[string]reflect.Value),
		services:     make(map[string]interface{}),
	}
}

func (sl *ServiceLocator) AddConfigPath(path string) {
	sl.configLoader.AddConfigPath(path)
}

func (sl *ServiceLocator) Set(name string, constructor interface{}) error {
	_, foundService := sl.services[name]
	_, foundConstructor := sl.constructors[name]
	if foundService || foundConstructor {
		err := errors.New("service already exists: " + name)
		if sl.panicMode {
			panic(err)
		}

		return err
	}

	constructorType := reflect.TypeOf(constructor)

	if constructorType.Kind() != reflect.Func {
		sl.services[name] = constructor

		return nil
	}

	if numOut := constructorType.NumOut(); (numOut > 2) || (numOut == 0) {
		err := errors.New("invalid count result elements: " + string(numOut) + " in constructor: " + name)
		if sl.panicMode {
			panic(err)
		}

		return err
	} else if (numOut == 2) && constructorType.Out(1).Kind() != reflect.Interface {
		err := errors.New("last result element must be error type in constructor:" + name)
		if sl.panicMode {
			panic(err)
		}

		return err
	}

	sl.constructors[name] = reflect.ValueOf(constructor)

	return nil
}

func (sl *ServiceLocator) Get(name string) (service interface{}, err error) {
	if service, found := sl.services[name]; found {
		return service, nil
	}

	if !sl.panicMode {
		defer func() {
			if exception := recover(); exception != nil {
				service = nil
				err = exception.(error)
			}
		}()
	}

	serviceConfig := sl.getConfigForService(name)

	constructor, found := sl.constructors[serviceConfig.Constructor]
	if !found {
		panic(errors.New("constructor not found for service: " + name))
	}

	var result []reflect.Value

	if len(serviceConfig.Arguments) == 0 {
		result = constructor.Call([]reflect.Value{})
	} else {
		result = constructor.Call(sl.prepareArguments(serviceConfig.Arguments))
	}

	switch len(result) {
	case 1:
		return result[0].Interface(), nil
	case 2:
		if result[1].Interface() != nil {
			panic(result[1].Interface().(error))
		}

		return result[0].Interface(), nil
	}

	panic(errors.New("invalid constructor: " + name))
}

func (sl *ServiceLocator) SetPanicMode(mode bool) {
	sl.panicMode = mode
}

func (sl *ServiceLocator) getConfig() iternalConfigMap {
	if sl.config != nil {
		return sl.config
	}

	if err := sl.configLoader.ReadInConfig(); err != nil {
		panic(err)
	}

	config := make(iternalConfigMap)

	if err := sl.configLoader.Marshal(&config); err != nil {
		panic(err)
	}

	sl.config = config

	return sl.config
}

func (sl *ServiceLocator) getConfigForService(name string) *iternalConfig {
	config := sl.getConfig()

	if result, found := config[name]; found {
		return &result
	}

	panic(errors.New("service: " + name + " not registered"))
}

func (sl *ServiceLocator) prepareArguments(arguments []interface{}) []reflect.Value {
	result := make([]reflect.Value, len(arguments))

	for i, argument := range arguments {
		value := reflect.ValueOf(argument)

		if value.Kind() == reflect.String {
			matches := dependencyRegexp.FindStringSubmatch(argument.(string))

			if len(matches) > 0 {
				if interfaceValue, err := sl.Get(matches[1]); err != nil {
					panic(err)
				} else {
					value = reflect.ValueOf(interfaceValue)
				}
			}
		}

		result[i] = value
	}

	return result
}
