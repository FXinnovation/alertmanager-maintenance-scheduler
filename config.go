package main

import (
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

// Config the configuration of the application
type Config struct {
	AlertmanagerURL string `yaml:"alertmanager_url"`
}

func loadConfig(path string) (*Config, error) {
	conf := &Config{}
	if path != "" {
		f, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal([]byte(f), &conf)
		if err != nil {
			return nil, err
		}
	}

	envURL := os.Getenv("ALERTMANAGER_URL")
	if envURL != "" {
		conf.AlertmanagerURL = envURL
	}

	log.Printf("Config loaded, path: %s", path)

	return conf, nil
}
