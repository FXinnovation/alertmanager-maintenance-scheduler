package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// Config the configuration of the application
type Config struct {
	AlertmanagerAPI string `yaml:"alertmanager_api"`
}

func loadConfig(path string) (conf *Config, err error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal([]byte(f), &conf)
	if err != nil {
		return nil, err
	}

	log.Printf("Config loaded, path: %s", path)

	return conf, nil
}
