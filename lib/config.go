package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/mitchellh/go-homedir"
	"github.com/vaughan0/go-ini"
)

type profiles map[string]map[string]string

type config interface {
	Parse() (profiles, error)
}

type fileConfig struct {
	file string
}

func NewConfigFromEnv() (config, error) {
	file := os.Getenv("AWS_CONFIG_FILE")
	if file == "" {
		home, err := homedir.Dir()
		if err != nil {
			return nil, err
		}
		file = filepath.Join(home, "/.aws/config")
		if _, err := os.Stat(file); os.IsNotExist(err) {
			file = ""
		}
	}
	return &fileConfig{file: file}, nil
}

func (c *fileConfig) Parse() (profiles, error) {
	if c.file == "" {
		return nil, nil
	}

	log.Debugf("Parsing config file %s", c.file)
	f, err := ini.LoadFile(c.file)
	if err != nil {
		return nil, fmt.Errorf("Error parsing config file %q: %v", c.file, err)
	}

	profiles := profiles{"okta": map[string]string{}}
	for sectionName, section := range f {
		profiles[strings.TrimPrefix(sectionName, "profile ")] = section
	}

	return profiles, nil
}

// sourceProfile returns either the defined source_profile or p if none exists
func sourceProfile(p string, from profiles) string {
	if conf, ok := from[p]; ok {
		if source := conf["source_profile"]; source != "" {
			return source
		}
	}
	return p
}

func GetValueFromProfile(profile string, from profiles, config_key string) (string, string, error) {
	for {
		config_value, ok := from[profile][config_key]
		if ok {
			return config_value, profile, nil
		}

		// Traverse up the chain of `source_profile`
		profile, ok = from[profile]["source_profile"]
		if !ok {
			break
		}
	}

	// Fallback to `okta` if no profile supplies the value
	profile = "okta"
	config_value, ok := from[profile][config_key]
	if ok {
		return config_value, profile, nil
	}

	return "", "", fmt.Errorf("Could not find %s in %s, parent profiles, or okta", config_key, profile)
}
