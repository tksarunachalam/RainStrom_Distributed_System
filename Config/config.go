package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

//read the cofig yaml and covert to go struct

type Config struct {
	Server []string `yaml:"server"`
	NodeId int      //self node id
	PeerId int
}

var Conf Config

func LoadConfig() (*Config, error) {
	// Read the YAML file
	data, err := ioutil.ReadFile("Config/config.yaml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &Conf)
	if err != nil {
		return nil, err
	}

	return &Conf, nil
}
