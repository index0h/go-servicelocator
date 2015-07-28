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
	configLoader *viper.Viper
	constructors map[string]func(...interface{}) (interface{}, error)
	services     map[string]interface{}
	config       iternalConfigMap
}

func New(fileName string, configType string) *ServiceLocator {
	configLoader := viper.New()
	configLoader.SetConfigType(configType)
	configLoader.SetConfigName(fileName)

	return &ServiceLocator{
		configLoader: configLoader,
		constructors: make(map[string]func(...interface{}) (interface{}, error)),
		services:     make(map[string]interface{}),
	}
}

func (sl *ServiceLocator) AddConfigPath(path string) {
	sl.configLoader.AddConfigPath(path)
}

func (sl *ServiceLocator) RegisterConstructor(
	name string,
	constructor func(...interface{}) (interface{}, error),
) error {
	if _, found := sl.constructors[name]; found {
		return errors.New("service already exists: " + name)
	}

	sl.constructors[name] = constructor

	return nil
}

func (sl *ServiceLocator) Get(name string) (service interface{}, err error) {
	var found bool

	if service, found = sl.services[name]; found {
		return service, nil
	}

	serviceConfig, err := sl.getConfigForService(name)
	if err != nil {
		return nil, err
	}

	constructor, found := sl.constructors[serviceConfig.Constructor]
	if !found {
		return nil, errors.New("constructor not found for service: " + name)
	}

	sl.prepareArguments(serviceConfig.Arguments)

	return constructor(serviceConfig.Arguments...)
}

func (sl *ServiceLocator) getConfig() (iternalConfigMap, error) {
	if sl.config != nil {
		return sl.config, nil
	}

	if err := sl.configLoader.ReadInConfig(); err != nil {
		return nil, err
	}

	config := make(iternalConfigMap)

	err := sl.configLoader.Marshal(&config)
	if err != nil {
		return nil, err
	}

	sl.config = config

	return sl.config, nil
}

func (sl *ServiceLocator) getConfigForService(name string) (*iternalConfig, error) {
	config, err := sl.getConfig()
	if err != nil {
		return nil, err
	}

	result, found := config[name]
	if !found {
		return nil, errors.New("service: " + name + " not registered")
	}

	return &result, nil
}

func (sl *ServiceLocator) prepareArguments(arguments []interface{}) (err error) {
	for i, argument := range arguments {
		if argument == nil {
			continue
		}

		if reflect.TypeOf(argument).Kind() != reflect.String {
			continue
		}

		if argument == "" {
			continue
		}

		matches := dependencyRegexp.FindStringSubmatch(argument.(string))
		if len(matches) == 0 {
			continue
		}

		arguments[i], err = sl.Get(matches[1])
		if err != nil {
			return err
		}
	}

	return nil
}
