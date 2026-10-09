package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"steam-cfg-portet/internal/notifier_service/domain"
	"time"
)

type NotifierUseCase struct {
	tgToken  string
	tgChatID string
	client   *http.Client
}

func NewNotifierUseCase(tgToken string, tgChatID string) domain.UseCase {
	return &NotifierUseCase{
		tgToken:  tgToken,
		tgChatID: tgChatID,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (n *NotifierUseCase) sendTelegramMessage(text string) error {
	if n.tgToken == "" || n.tgChatID == "" {
		return errors.New("telegram credentials are empty")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.tgToken)

	payload := map[string]string{
		"chat_id":    n.tgChatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tg payload: %v", err)
	}

	resp, err := n.client.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to dispatch http post to telegram: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned bad status code: %d", resp.StatusCode)
	}

	return nil
}

func (n *NotifierUseCase) OnConfigInstalled(ctx context.Context, event *domain.ConfigInstalledEvent) error {
	if event == nil {
		return nil
	}

	log.Printf("[NOTIFICATION] Config Installed! User ID: %d | Game: %s | Config ID: %s | Name: %s",
		event.UserID, event.Game, event.ID, event.Name)

	msgText := fmt.Sprintf("🎮 *Config Installed!*\n\n• *Game:* %s\n• *Preset:* %s\n• *User ID:* %d",
		event.Game, event.Name, event.UserID)

	return n.sendTelegramMessage(msgText)
}

func (n *NotifierUseCase) OnConfigSaved(ctx context.Context, event *domain.ConfigSavedEvent) error {
	if event == nil {
		return nil
	}

	log.Printf("[NOTIFICATION] New Config Saved in Cloud! Game: %s | Config ID: %s | Name: %s",
		event.Game, event.ID, event.Name)

	msgText := fmt.Sprintf("☁️ *New Config Saved in Cloud!*\n\n• *Game:* %s\n• *Preset Name:* %s\n• *ID:* `%s`",
		event.Game, event.Name, event.ID)

	return n.sendTelegramMessage(msgText)
}

func (n *NotifierUseCase) OnConfigDeleted(ctx context.Context, event *domain.ConfigDeletedEvent) error {
	if event == nil {
		return nil
	}

	log.Printf("[NOTIFICATION] Config Deleted by Author! Game: %s | Name: %s | Author: %s",
		event.Game, event.Name, event.AuthorNickName)

	msgText := fmt.Sprintf("🗑 *Config Deleted from Cloud!*\n\n• *Game:* %s\n• *Preset:* %s\n• *Author:* %s",
		event.Game, event.Name, event.AuthorNickName)

	return n.sendTelegramMessage(msgText)
}
