package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nodewee/llm-caller/pkg/utils"

	"github.com/spf13/viper"
)

const (
	// DefaultConfigDir is the default directory for configuration files
	DefaultConfigDir = ".llm-caller"
	// ConfigFile is the name of the configuration file
	ConfigFile = "config"
	// ConfigType is the type of the configuration file
	ConfigType = "yaml"
)

// Configuration keys
const (
	KeyTemplateDir = "template_dir"
	KeySecretFile  = "secret_file"
)

// Config manages the application configuration
type Config struct {
	viper *viper.Viper
}

// New creates a new config instance
func New() (*Config, error) {
	v := viper.New()

	// Set defaults using cross-platform path handling
	configDir, err := utils.GetUserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config directory: %w", err)
	}

	v.SetDefault(KeyTemplateDir, filepath.Join(configDir, "templates"))
	v.SetDefault(KeySecretFile, filepath.Join(configDir, "keys.json"))

	// Setup config file with cross-platform directory permissions
	if err := utils.CreateDirWithPlatformPermissions(configDir); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	v.SetConfigName(ConfigFile)
	v.SetConfigType(ConfigType)
	v.AddConfigPath(configDir)

	// Try to read the config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error occurred
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, create it
		configFile := filepath.Join(configDir, ConfigFile+"."+ConfigType)
		if err := v.WriteConfigAs(configFile); err != nil {
			return nil, fmt.Errorf("failed to create config file: %w", err)
		}
	}

	return &Config{viper: v}, nil
}

// Get returns the value associated with the key
func (c *Config) Get(key string) interface{} {
	return c.viper.Get(key)
}

// GetString returns the value associated with the key as a string
func (c *Config) GetString(key string) string {
	return c.viper.GetString(key)
}

// Set sets the value for the key
func (c *Config) Set(key string, value interface{}) error {
	c.viper.Set(key, value)
	return c.viper.WriteConfig()
}

// List returns all the configuration settings
func (c *Config) List() map[string]interface{} {
	return c.viper.AllSettings()
}

// Delete removes the value for the key
func (c *Config) Delete(key string) error {
	// Get the current config file used
	configFile := c.viper.ConfigFileUsed()
	if configFile == "" {
		// If no config file is used, construct the path
		configDir, err := utils.GetUserConfigDir()
		if err != nil {
			return fmt.Errorf("failed to get user config directory: %w", err)
		}
		configFile = filepath.Join(configDir, ConfigFile+"."+ConfigType)
	}

	// Read the existing config file to check if the key exists in the file
	data, err := os.ReadFile(configFile)
	if err != nil {
		// If file doesn't exist, the key doesn't exist in user configuration
		return fmt.Errorf("key %s not found in configuration", key)
	}

	// Parse the existing config
	existingConfig := make(map[string]interface{})
	if len(data) > 0 {
		tempViper := viper.New()
		tempViper.SetConfigType(ConfigType)
		if err := tempViper.ReadConfig(bytes.NewBuffer(data)); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
		existingConfig = tempViper.AllSettings()
	}

	// Check if the key exists in the user configuration (not just defaults)
	if _, exists := existingConfig[key]; !exists {
		return fmt.Errorf("key %s not found in configuration", key)
	}

	// Remove the key from the configuration
	delete(existingConfig, key)

	// Create a new viper instance and set only the remaining user configuration
	newViper := viper.New()
	for k, v := range existingConfig {
		newViper.Set(k, v)
	}

	// Set config file info
	newViper.SetConfigName(ConfigFile)
	newViper.SetConfigType(ConfigType)
	newViper.AddConfigPath(filepath.Dir(configFile))

	// Write the updated configuration
	if err := newViper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Reload the current viper instance by re-reading the config
	if err := c.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	return nil
}

// GetConfigFilePath returns the path to the configuration file
func (c *Config) GetConfigFilePath() string {
	configDir, err := utils.GetUserConfigDir()
	if err != nil {
		return ""
	}

	return filepath.Join(configDir, ConfigFile+"."+ConfigType)
}

// GetDefaultTemplateDir returns the default template directory path
func GetDefaultTemplateDir() (string, error) {
	configDir, err := utils.GetUserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	return filepath.Join(configDir, "templates"), nil
}

// EnsureTemplateDir ensures the template directory exists and returns its path
func (c *Config) EnsureTemplateDir() (string, error) {
	templateDir := c.GetString(KeyTemplateDir)
	if err := utils.CreateDirWithPlatformPermissions(templateDir); err != nil {
		return "", fmt.Errorf("failed to create template directory: %w", err)
	}
	return templateDir, nil
}
