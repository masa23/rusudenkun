package config

import (
	"errors"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// SakuraAIEngineの設定
type configSakuraAIEngine struct {
	URL        string        `yaml:"URL"`
	Token      string        `yaml:"Token"`
	Model      string        `yaml:"Model"`
	Timeout    time.Duration `yaml:"Timeout"`
	MaxRetries int           `yaml:"MaxRetries"`
}

// Slackの設定
type configSlack struct {
	WebHookURL string `yaml:"WebHookURL"`
}

// アプリケーション全体の設定
type Config struct {
	Name                string               `yaml:"Name"`
	URL                 string               `yaml:"URL"`
	WebsocketURL        string               `yaml:"WebsocketURL"`
	WebsocketOrigin     string               `yaml:"WebsocketOrigin"`
	MessageSound        string               `yaml:"MessageSound"`
	RecordingTimeout    time.Duration        `yaml:"RecordingTimeout"`
	RecordingMaxSilence time.Duration        `yaml:"RecordingMaxSilence"`
	LogFile             string               `yaml:"LogFile"`
	SakuraAIEngine      configSakuraAIEngine `yaml:"SakuraAIEngine"`
	Slack               configSlack          `yaml:"Slack"`
}

// 設定のバリデーションを行う
func (c *Config) validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}
	if c.URL == "" {
		return errors.New("url is required")
	}
	if c.WebsocketURL == "" {
		return errors.New("websocketURL is required")
	}
	if c.RecordingTimeout == 0 {
		c.RecordingTimeout = 60 * time.Second
	}
	if c.RecordingMaxSilence == 0 {
		c.RecordingMaxSilence = 20 * time.Second
	}

	return nil
}

// 環境変数で設定を上書きする
func (c *Config) envOverWrite() {
	if v := os.Getenv("ARI_NAME"); v != "" {
		c.Name = v
	}
	if v := os.Getenv("ARI_URL"); v != "" {
		c.URL = v
	}
	if v := os.Getenv("ARI_WEBSOCKET_URL"); v != "" {
		c.WebsocketURL = v
	}
	if v := os.Getenv("ARI_WEBSOCKET_ORIGIN"); v != "" {
		c.WebsocketURL = v
	}
}

// 設定ファイルを読み込む
func Load(path string) (*Config, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf Config
	err = yaml.Unmarshal(buf, &conf)
	if err != nil {
		return nil, err
	}

	log.Printf("config %+v", conf)

	conf.envOverWrite()

	// 設定のバリデーション
	if err := conf.validate(); err != nil {
		return nil, err
	}

	return &conf, nil
}
