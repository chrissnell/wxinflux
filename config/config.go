package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config is the main configuration object
type Config struct {
	InfluxDB InfluxDBConfig `yaml:"influxdb"`
	Si2000   Si2000Config   `yaml:"si2000"`
}

// InfluxDBConfig holds InfluxDB-specific configuration
type InfluxDBConfig struct {
	InfluxURL    string `yaml:"url"`
	InfluxDBName string `yaml:"dbname"`
	InfluxUser   string `yaml:"user"`
	InfluxPass   string `yaml:"pass"`
}

// Si2000Config holds Si2000-specific configuration
type Si2000Config struct {
	Device string `yaml:"device"`
	Baud   uint16 `yaml:"baud"`
}

// New creates an new config object from the given filename.
func New(filename string) (Config, error) {
	cfgFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	c := Config{}
	err = yaml.Unmarshal(cfgFile, &c)
	if err != nil {
		return Config{}, err
	}
	return c, nil
}
