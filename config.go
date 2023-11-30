package grpcx

import (
	"github.com/spf13/viper"

	"github.com/autom8ter/grpcx/internal/utils"
)

// ConfigDefaults are the default values for the config file
var ConfigDefaults = map[string]interface{}{
	`api.port`:                   8080,
	`logging.level`:              `debug`,
	`logging.tags`:               []string{`method`, `context_id`, `error`},
	`logging.request_body`:       false,
	`database.migrate`:           true,
	`api.cors.enabled`:           false,
	`api.cors.allowed_origins`:   []string{`*`},
	`api.cors.allowed_methods`:   []string{`GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`, `PATCH`},
	`api.cors.allowed_headers`:   []string{`*`},
	`api.cors.exposed_headers`:   []string{`*`},
	`api.cors.allow_credentials`: true,
}

// LoadConfig loads a config file from the given path(if it exists) and sets the ConfigDefaults
func LoadConfig(apiName, filePath, envPrefix string) (*viper.Viper, error) {
	v := viper.New()
	if filePath != "" {
		v.SetConfigFile(filePath)
		if err := v.ReadInConfig(); err != nil {
			return nil, utils.WrapError(err, "failed to read config file")
		}
	}
	if envPrefix != "" {
		v.SetEnvPrefix(envPrefix)
		v.AutomaticEnv()
	}
	v.SetDefault("api.name", apiName)
	for k, val := range ConfigDefaults {
		v.SetDefault(k, val)
	}
	return v, nil
}
