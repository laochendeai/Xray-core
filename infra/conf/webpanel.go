package conf

import (
	"github.com/xtls/xray-core/app/webpanel"
	"github.com/xtls/xray-core/common/errors"
)

// WebPanelConfig is the JSON configuration for the web panel.
type WebPanelConfig struct {
	Listen      string `json:"listen"`
	APIEndpoint string `json:"apiEndpoint"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	JWTSecret   string `json:"jwtSecret"`
	ConfigPath  string `json:"configPath"`
	CertFile    string `json:"certFile"`
	KeyFile     string `json:"keyFile"`
}

// Build implements Buildable.
func (c *WebPanelConfig) Build() (*webpanel.Config, error) {
	if c.Listen == "" && c.APIEndpoint == "" {
		return nil, errors.New("Web panel must have a listen address or API endpoint.")
	}

	return &webpanel.Config{
		Listen:      c.Listen,
		ApiEndpoint: c.APIEndpoint,
		Username:    c.Username,
		Password:    c.Password,
		JwtSecret:   c.JWTSecret,
		ConfigPath:  c.ConfigPath,
		CertFile:    c.CertFile,
		KeyFile:     c.KeyFile,
	}, nil
}
