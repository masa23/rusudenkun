package audiotrans

import (
	"context"
	"fmt"

	"github.com/masa23/aiengine-go"
	"github.com/masa23/rusudenkun/config"
)

// Client はAI Engineクライアントです
type Client struct {
	client *aiengine.Client
	config config.Config
}

// NewClient は新しいAI Engineクライアントを作成します
func NewClient(cfg config.Config) (*Client, error) {
	// 環境変数からAPIキーを取得する場合
	if cfg.SakuraAIEngine.Token == "" {
		client, err := aiengine.NewClientFromEnv(
			aiengine.WithBaseURL(cfg.SakuraAIEngine.URL),
			aiengine.WithTimeout(cfg.SakuraAIEngine.Timeout),
			aiengine.WithMaxRetries(cfg.SakuraAIEngine.MaxRetries),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create AI Engine client from env: %w", err)
		}
		return &Client{client: client, config: cfg}, nil
	}

	// APIキーを直接指定する場合
	client := aiengine.NewClient(cfg.SakuraAIEngine.Token,
		aiengine.WithBaseURL(cfg.SakuraAIEngine.URL),
		aiengine.WithTimeout(cfg.SakuraAIEngine.Timeout),
		aiengine.WithMaxRetries(cfg.SakuraAIEngine.MaxRetries),
	)
	return &Client{client: client, config: cfg}, nil
}

// AudioTranscription は音声ファイルをテキストに変換します
func (c *Client) AudioTranscription(audioFile string) (string, error) {
	// 音声書き起こしリクエストを作成
	req := &aiengine.TranscriptionRequest{
		File:  audioFile,
		Model: c.config.SakuraAIEngine.Model,
	}

	// 音声書き起こしを実行
	ctx, cancel := context.WithTimeout(context.Background(), c.config.SakuraAIEngine.Timeout)
	defer cancel()

	resp, err := c.client.CreateTranscription(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create transcription: %w", err)
	}

	return resp.Text, nil
}
