package config

import (
	"reflect"

	"github.com/flashbots/bmonitor/utils"
)

type Config struct {
	Log     *Log     `yaml:"log"`
	Monitor *Monitor `yaml:"monitor"`
	Server  *Server  `yaml:"server"`
}

func New() *Config {
	return &Config{
		Log:     &Log{},
		Monitor: &Monitor{},
		Server:  &Server{},
	}
}

func (c *Config) Validate() error {
	return validate(c)
}

func validate(item interface{}) error {
	v := reflect.ValueOf(item)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	errs := []error{}
	for idx := 0; idx < v.NumField(); idx++ {
		field := v.Field(idx)

		if field.Kind() == reflect.Ptr && field.IsNil() {
			continue
		}

		if v, ok := field.Interface().(validatee); ok {
			if err := v.Validate(); err != nil {
				errs = append(errs, err)
			}
		}

		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.Struct:
			if err := validate(field.Interface()); err != nil {
				errs = append(errs, err)
			}
		case reflect.Slice, reflect.Array:
			for jdx := 0; jdx < field.Len(); jdx++ {
				if err := validate(field.Index(jdx).Interface()); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	return utils.FlattenErrors(errs)
}
