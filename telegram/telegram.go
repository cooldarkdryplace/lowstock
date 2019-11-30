package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/cooldarkdryplace/lowstock"

	"github.com/VictoriaMetrics/metrics"
)

const (
	baseURL           = "https://api.telegram.org/bot"
	methodSendMessage = "sendMessage"
	methodGetUpdates  = "getUpdates"

	// Wait timeout for longpolling
	timeout = 60
)

var (
	updSuccessCounter = metrics.NewCounter(`tg_api_calls{status="success", method="getUpdates"}`)
	updFailureCounter = metrics.NewCounter(`tg_api_calls{status="failure", method="getUpdates"}`)

	msgSuccessCounter = metrics.NewCounter(`tg_api_calls{status="success", method="sendMessage"}`)
	msgFailureCounter = metrics.NewCounter(`tg_api_calls{status="failure", method="sendMessage"}`)
)

type Telegram struct {
	token string
}

func New(t string) *Telegram {
	return &Telegram{token: t}
}

type UpdatesResponse struct {
	Ok      bool     `json:"ok"`
	Updates []Update `json:"result"`
}

func toMessengerUpdate(u Update) lowstock.MessengerUpdate {
	return lowstock.MessengerUpdate{
		ID:      u.ID,
		Command: u.Command(),
		Text:    u.Text(),
		ChatID:  u.ChatID(),
		UserID:  u.UserID(),
	}
}

func toMessengerUpdates(tu []Update) []lowstock.MessengerUpdate {
	updates := make([]lowstock.MessengerUpdate, 0, len(tu))

	for _, upd := range tu {
		updates = append(updates, toMessengerUpdate(upd))
	}

	return updates
}

// Updates provide Telegram Bot updates with IDs greater than provided value.
func (t *Telegram) Updates(lastMsgID int64) ([]lowstock.MessengerUpdate, error) {
	url := fmt.Sprintf("%s%s/%s?timeout=%d&offset=%d", baseURL, t.token, methodGetUpdates, timeout, lastMsgID)

	apiResponse := &UpdatesResponse{}

	r, err := http.Get(url)
	if err != nil {
		updFailureCounter.Inc()
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		updFailureCounter.Inc()
		return nil, fmt.Errorf("unexpected status code: %s", r.Status)
	}

	if err = json.NewDecoder(r.Body).Decode(apiResponse); err != nil {
		updFailureCounter.Inc()
		return nil, fmt.Errorf("failed to unmarshal updates: %w", err)
	}

	updSuccessCounter.Inc()
	return toMessengerUpdates(apiResponse.Updates), nil
}

type InlineKeyboardButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type SendMessageRequest struct {
	ChatID                int64                 `json:"chat_id"`
	Text                  string                `json:"text"`
	ParseMode             string                `json:"parse_mode"`
	DisableWebPagePreview bool                  `json:"disable_web_page_preview"`
	DisableNotification   bool                  `json:"disable_notification"`
	ReplyToMessageID      *int64                `json:"reply_to_message_id,omitempty"`
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

func (t *Telegram) sendMessage(msg SendMessageRequest) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	apiURL := fmt.Sprintf("%s%s/%s", baseURL, t.token, methodSendMessage)

	resp, err := http.Post(apiURL, "application/json", bytes.NewReader(data))
	if err != nil {
		msgFailureCounter.Inc()
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msgFailureCounter.Inc()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("failed to send message, status: %s, body: %s", resp.Status, string(body))
	}

	msgSuccessCounter.Inc()

	return nil
}

// SendTextMessage to the chat with provided ID.
func (t *Telegram) SendTextMessage(text string, chatID int64) error {
	msg := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	return t.sendMessage(msg)
}

func (t *Telegram) SendLoginURL(text, uri string, chatID int64) error {
	btn := InlineKeyboardButton{
		Text: "Login to Etsy",
		URL:  uri,
	}

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			[]InlineKeyboardButton{btn},
		},
	}

	msg := SendMessageRequest{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "Markdown",
		ReplyMarkup: keyboard,
	}

	return t.sendMessage(msg)
}

type Chat struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	UserName  string `json:"username"`
}

type Entity struct {
	Type   string `json:"type"`
	Offset int64  `json:"offset"`
	Length int64  `json:"length"`
}

type Message struct {
	ID       int64    `json:"message_id"`
	Date     int      `json:"date"`
	Chat     Chat     `json:"chat"`
	Entities []Entity `json:"entities"`
	Text     string   `json:"text"`
	From     User     `json:"from"`
}

type Update struct {
	ID      int64   `json:"update_id"`
	Message Message `json:"message"`
}

func (u Update) Type() string {
	if len(u.Message.Entities) == 0 {
		return ""
	}

	return u.Message.Entities[0].Type
}

func extractCommand(text string) string {
	return strings.Split(text, " ")[0]
}

func (u Update) Command() string {
	if t := u.Type(); t != "bot_command" {
		log.Println("Not a command, ignoring")
		return ""
	}

	command := extractCommand(u.Message.Text)

	return command
}

func (u Update) ChatID() int64 {
	return u.Message.Chat.ID
}

func (u Update) UserID() int64 {
	return u.Message.From.ID
}

func (u Update) Text() string {
	return u.Message.Text
}
