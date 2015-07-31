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
	logger LoggerInterface
	configLoader *viper.Viper
	constructors map[string]reflect.Value
	services     map[string]interface{}
	config       iternalConfigMap
}

func New(fileName string) *ServiceLocator {
	configLoader := viper.New()
	configLoader.SetConfigType("yaml")
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

func (sl *ServiceLocator) Get(name string) (service interface{}, err error) {
	sl.debug("Get service: " + name)

	if service, found := sl.services[name]; found {
		return service, nil
	}

	sl.debug("Build service: " + name)

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
		sl.panic(errors.New("constructor not found for service: " + name))
	}

	var result []reflect.Value

	if len(serviceConfig.Arguments) == 0 {
		result = constructor.Call([]reflect.Value{})
	} else {
		result = constructor.Call(sl.prepareArguments(serviceConfig.Arguments))
	}

	switch len(result) {
	case 1:
		sl.debug("Service ok: " + name)

		return result[0].Interface(), nil
	case 2:
		if result[1].Interface() != nil {
			sl.panic(result[1].Interface().(error))
		}

		sl.debug("Service ok: " + name)

		return result[0].Interface(), nil
	}

	return nil, sl.error(errors.New("invalid constructor: " + name))
}

func (sl *ServiceLocator) SetConstructor(name string, constructor interface{}) (err error) {
	constructorType := reflect.TypeOf(constructor)

	if constructorType.Kind() != reflect.Func {
		sl.panic(errors.New("constructor must be func: " + name))
	}

	if numOut := constructorType.NumOut(); (numOut > 2) || (numOut == 0) {
		sl.panic(errors.New("invalid count result elements: " + string(numOut) + " in constructor: " + name))
	} else if (numOut == 2) && constructorType.Out(1).Kind() != reflect.Interface {
		sl.panic(errors.New("last result element must be error type in constructor:" + name))
	}

	if _, foundService := sl.services[name]; foundService {
		err = sl.error(errors.New("service already exists: " + name))
	}

	if _, foundConstructor := sl.constructors[name]; foundConstructor {
		err = sl.error(errors.New("constructor already exists: " + name))
	}

	sl.constructors[name] = reflect.ValueOf(constructor)

	return err
}

func (sl *ServiceLocator) SetService(name string, service interface{}) {
	if _, foundService := sl.services[name]; foundService {
		sl.warning(errors.New("service already exists: " + name))
	}

	sl.services[name] = service
}

func (sl *ServiceLocator) SetConfig(serviceName string, constructorName string, arguments []interface{}) {
	if _, foundService := sl.services[serviceName]; foundService {
		sl.warning(errors.New("service already exists: " + serviceName))
	}

	if _, foundConfig := sl.config[serviceName]; foundConfig {
		sl.warning(errors.New("config already exists: " + serviceName))
	}

	sl.config[serviceName] = iternalConfig{Constructor:constructorName, Arguments:arguments}
}

func (sl *ServiceLocator) SetPanicMode(mode bool) {
	sl.panicMode = mode
}

func (sl *ServiceLocator) SetLogger(logger LoggerInterface) {
	sl.logger = logger
}

func (sl *ServiceLocator) SetConfigType(configType string) {
	sl.configLoader.SetConfigType(configType)
}

func (sl *ServiceLocator) getConfig() iternalConfigMap {
	if sl.config != nil {
		return sl.config
	}

	if err := sl.configLoader.ReadInConfig(); err != nil {
		sl.panic(err)
	}

	config := make(iternalConfigMap)

	if err := sl.configLoader.Marshal(&config); err != nil {
		sl.panic(err)
	}

	sl.config = config

	return sl.config
}

func (sl *ServiceLocator) getConfigForService(name string) *iternalConfig {
	config := sl.getConfig()

	if result, found := config[name]; found {
		return &result
	}

	sl.panic(errors.New("service: " + name + " not registered"))

	return nil
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


func (sl *ServiceLocator) panic(err error) {
	if sl.logger != nil {
		sl.logger.Fatal(err.Error())
	}

	panic(err)
}

func (sl *ServiceLocator) error(err error) error {
	if sl.logger != nil {
		sl.logger.Error(err.Error())
	}

	if sl.panicMode {
		panic(err)
	}

	return err
}

func (sl *ServiceLocator) warning(err error) error {
	if sl.logger != nil {
		sl.logger.Warn(err.Error())
	}

	return err
}

func (sl *ServiceLocator) debug(message string) {
	if sl.logger != nil {
		sl.logger.Debug(message)
	}
}
