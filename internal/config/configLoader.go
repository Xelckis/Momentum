package config

import (
	"github.com/BurntSushi/toml"
	"log"
)

type Config struct {
	Server   server
	Security security
}

type server struct {
	HttpPort  string `toml:"http_port"`
	HttpsPort string `toml:"https_port"`

	RedirectToHttps bool `toml:"redirect_to_https"`

	ReadTimeout  int  `toml:"read_timeout"`
	WriteTimeout int  `toml:"write_timeout"`
	IdleTimeout  int  `toml:"idle_timeout"`
	LogEndpoint  bool `toml:"log_endpoint_enabled"`
}

type security struct {
	CertFile                 string `toml:"cert_file"`
	KeyFile                  string `toml:"key_file"`
	PreferServerCipherSuites bool   `toml:"prefer_server_cipher_suites"`
	TlsVersion               string `toml:"min_tls_version"`

	AllowedOrigins []string `toml:"allowed_origins"`
}

func (c *Config) LoadConfig() {
	_, err := toml.DecodeFile("config/config.toml", &c)
	if err != nil {
		log.Fatal("Cannot load config file: %w", err)
	}

}
