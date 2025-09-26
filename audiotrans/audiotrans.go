package audiotrans

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

func AudioTranscription(url string, token string, model string, audioFile string) (text string, err error) {
	f, err := os.Open(audioFile)
	if err != nil {
		return "", fmt.Errorf("open audio file: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if err := w.WriteField("model", model); err != nil {
		return "", fmt.Errorf("write model field: %w", err)
	}

	fw, err := w.CreateFormFile("file", f.Name())
	if err != nil {
		return "", fmt.Errorf("create file field: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", fmt.Errorf("copy audio: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("finalize multipart: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slurp, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("bad status %d: %s", resp.StatusCode, string(slurp))
	}

	var out struct {
		Model string `json:"model"`
		Text  string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return out.Text, nil
}
