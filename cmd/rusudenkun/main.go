package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CyCoreSystems/ari/v6"
	"github.com/CyCoreSystems/ari/v6/client/native"
	"github.com/CyCoreSystems/ari/v6/ext/record"
	"github.com/masa23/rusudenkun/audiotrans"
	"github.com/masa23/rusudenkun/config"
	"github.com/slack-go/slack"
	"golang.org/x/exp/slog"
)

var log = slog.New(slog.NewJSONHandler(os.Stderr, nil))
var conf *config.Config
var aiClient *audiotrans.Client

func main() {
	var confPath string
	var err error
	flag.StringVar(&confPath, "config", "config.yaml", "path to config file")
	flag.Parse()

	conf, err = config.Load(confPath)
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// AI Engineクライアントを初期化
	aiClient, err = audiotrans.NewClient(*conf)
	if err != nil {
		log.Error("failed to create AI Engine client", "error", err)
		os.Exit(1)
	}

	logfd, err := os.OpenFile(conf.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error("failed to open log file", "error", err)
		os.Exit(1)
	}
	defer logfd.Close()

	log = slog.New(slog.NewJSONHandler(logfd, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cl, err := native.Connect(&native.Options{
		Application:     "rusudenkun",
		Logger:          log.With("app", "rusudenkun"),
		Username:        "asterisk",
		Password:        "asterisk",
		URL:             conf.URL,
		WebsocketURL:    conf.WebsocketURL,
		WebsocketOrigin: conf.WebsocketOrigin,
	})
	if err != nil {
		log.Error("failed to connect", "error", err)
		os.Exit(1)
	}
	defer cl.Close()

	log.Info("connected to asterisk")

	sub := cl.Bus().Subscribe(nil, "StasisStart")

	for {
		select {
		case e := <-sub.Events():
			v := e.(*ari.StasisStart)
			log.Info("StasisStart", "channel", v.Channel.ID)
			go app(ctx, cl.Channel().Get(v.Key(ari.ChannelKey, v.Channel.ID)), aiClient)

		case <-ctx.Done():
			return
		}
	}
}

func announce(ctx context.Context, h *ari.ChannelHandle) {
	if conf.MessageSound == "" {
		log.Info("no announce sound configured")
		return
	}

	media := fmt.Sprintf("sound:%s", conf.MessageSound)
	pb, err := h.Play("announce-"+h.ID(), media)
	if err != nil {
		log.Error("failed to start announce", "error", err, "media", media)
		return
	}

	log.Info("playing announce", "media", media)

	// 再生の終了/失敗/停止を待つ
	sub := pb.Subscribe(
		ari.Events.PlaybackFinished,
	)
	defer sub.Cancel()

	for {
		select {
		case <-ctx.Done():
			log.Warn("announce canceled by context")
			_ = pb.Stop()
			return
		case ev := <-sub.Events():
			switch ev.(type) {
			case *ari.PlaybackFinished:
				log.Info("announce finished")
			}
			return
		}
	}
}

func recording(ctx context.Context, h *ari.ChannelHandle) (path string, err error) {
	res, err := record.Record(ctx, h,
		record.TerminateOn("any"),
		record.IfExists("overwrite"),
		record.WithLogger(log.With("app", "recorder")),
		record.Beep(),
		record.Format("wav"),
		record.MaxDuration(conf.RecordingTimeout),
		record.MaxSilence(conf.RecordingMaxSilence),
	).Result()
	if err != nil {
		log.Error("failed to record", "error", err)
		return
	}

	path = fmt.Sprintf("rusudenkun-%s.wav", h.ID())
	if err = res.Save(path); err != nil {
		log.Error("failed to save recording", "error", err)
	}

	log.Info("completed recording")

	return
}
func app(ctx context.Context, h *ari.ChannelHandle, client *audiotrans.Client) {
	defer h.Hangup()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Info("Running app", "channel", h.ID())

	end := h.Subscribe(ari.Events.StasisEnd)
	defer end.Cancel()

	go func() {
		<-end.Events()
		cancel()
	}()

	if err := h.Answer(); err != nil {
		log.Error("failed to answer call", "error", err)
		return
	}
	time.Sleep(150 * time.Millisecond)

	announce(ctx, h)

	path, err := recording(ctx, h)
	if err != nil {
		log.Error("failed to record", "error", err)
		return
	}

	h.Hangup()
	time.Sleep(1 * time.Second)

	log.Info("recorded", "path", path)
	path = filepath.Join("/var/spool/asterisk/recording/", path)
	text, err := aiClient.AudioTranscription(path)
	if err != nil {
		log.Error("failed to transcribe audio", "error", err)
		return
	}

	log.Info("transcribed text", "text", text)

	if err := sendMessage(conf.Slack.WebHookURL, text); err != nil {
		log.Error("failed to send Slack message", "error", err)
	}
}

func sendMessage(webhookURL, text string) error {
	attachment := slack.Attachment{
		Color: "good",
		Fields: []slack.AttachmentField{
			{
				Title: "時刻",
				Value: time.Now().Format("2006-01-02 15:04:05"),
				Short: true,
			},
			{
				Title: "メッセージ（文字起こし）",
				Value: text,
			},
		},
	}

	msg := slack.WebhookMessage{
		Attachments: []slack.Attachment{attachment},
	}

	return slack.PostWebhook(webhookURL, &msg)
}
