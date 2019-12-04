package telegram

import (
	"testing"

	"github.com/cooldarkdryplace/lowstock"

	"github.com/google/go-cmp/cmp"
)

func TestToMessengerUpdate(t *testing.T) {
	var (
		id      int64 = 42
		chatID  int64 = 13
		userID  int64 = 2546
		command       = "/test"
		text          = "/test"
	)

	expectedMsgUpd := lowstock.MessengerUpdate{
		ID:      id,
		ChatID:  chatID,
		UserID:  userID,
		Command: command,
		Text:    text,
	}

	input := Update{
		ID: id,
		Message: Message{
			Entities: []Entity{
				Entity{Type: "bot_command"},
			},
			Chat: Chat{
				ID: chatID,
			},
			From: User{
				ID: userID,
			},
			Text: text,
		},
	}

	actualMsgUpd := toMessengerUpdate(input)

	if diff := cmp.Diff(expectedMsgUpd, actualMsgUpd); diff != "" {
		t.Errorf("Updates do not match:\n%s", diff)
	}

}
