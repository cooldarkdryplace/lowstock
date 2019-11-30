package lowstock

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDoPin(t *testing.T) {
	var (
		expectedPin              = "42"
		expectedChatID     int64 = 100500
		expectedUserID     int64 = 9500
		expectedEtsyUserID int64 = 5432
		initialToken             = "tmp_token"
		initialTokenSecret       = "tmp_token_secret"
		finalToken               = "final_token"
		finalTokenSecret         = "final_token_secret"
	)

	expectedUser := User{
		EtsyUserID:  expectedEtsyUserID,
		ChatUserID:  expectedUserID,
		ChatID:      expectedChatID,
		Token:       finalToken,
		TokenSecret: finalTokenSecret,
	}

	storage := &StorageMock{
		TokenDetailsFunc: func(ctx context.Context, id int64) (TokenDetails, error) {
			if id != expectedUserID {
				t.Errorf("Got user ID: %d, expected: %d", id, expectedUserID)
			}

			return TokenDetails{
				Token:       initialToken,
				TokenSecret: initialTokenSecret,
			}, nil
		},
		SaveUserFunc: func(ctx context.Context, user User) error {
			if diff := cmp.Diff(user, expectedUser); diff != "" {
				t.Errorf("Users are different:\n%s", diff)
			}

			return nil
		},
	}

	messenger := &MessengerMock{
		SendTextMessageFunc: func(msg string, chatID int64) error {
			if chatID != expectedChatID {
				t.Errorf("Got chat ID: %d, expected: %d", chatID, expectedChatID)
			}

			return nil
		},
	}

	etsy := &EtsyMock{
		CallbackFunc: func(ctx context.Context, pin, token, secret string) (TokenDetails, error) {
			if pin != expectedPin {
				t.Errorf("Got pin: %s, expected: %s", pin, expectedPin)
			}

			if token != initialToken {
				t.Errorf("Got token: %s, expected: %s", token, initialToken)
			}

			if secret != initialTokenSecret {
				t.Errorf("Got secret: %s, expected: %s", secret, initialTokenSecret)
			}

			return TokenDetails{
				Token:       finalToken,
				TokenSecret: finalTokenSecret,
			}, nil
		},
		UserIDFunc: func(accessToken, accessSecret string) (int64, error) {
			return expectedEtsyUserID, nil
		},
	}

	ls := New(etsy, messenger, storage)

	update := MessengerUpdate{
		Command: "/pin",
		Text:    "/pin " + expectedPin,
		ChatID:  expectedChatID,
		UserID:  expectedUserID,
	}

	err := ls.DoPin(context.Background(), update)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

type EtsyMock struct {
	CallbackFunc func(ctx context.Context, pin, token, secret string) (TokenDetails, error)
	LoginFunc    func(ctx context.Context, id int64) (string, TokenDetails, error)
	UserIDFunc   func(accessToken, accessSecret string) (int64, error)
	UpdatesFunc  func(ctx context.Context) ([]Update, error)
}

func (e *EtsyMock) Callback(ctx context.Context, pin, token, secret string) (TokenDetails, error) {
	return e.CallbackFunc(ctx, pin, token, secret)
}

func (e *EtsyMock) Login(ctx context.Context, id int64) (string, TokenDetails, error) {
	return e.LoginFunc(ctx, id)
}

func (e *EtsyMock) UserID(accessToken, accessSecret string) (int64, error) {
	return e.UserIDFunc(accessToken, accessSecret)
}

func (e *EtsyMock) Updates(ctx context.Context) ([]Update, error) {
	return e.UpdatesFunc(ctx)
}

type MessengerMock struct {
	SendLoginURLFunc    func(text, url string, chatID int64) error
	SendTextMessageFunc func(msg string, chatID int64) error
	UpdatesFunc         func(lastMsgID int64) ([]MessengerUpdate, error)
}

func (m *MessengerMock) SendLoginURL(text, url string, chatID int64) error {
	return m.SendLoginURLFunc(text, url, chatID)
}

func (m *MessengerMock) SendTextMessage(msg string, chatID int64) error {
	return m.SendTextMessageFunc(msg, chatID)
}

func (m *MessengerMock) Updates(lastMsgID int64) ([]MessengerUpdate, error) {
	return m.UpdatesFunc(lastMsgID)
}

type StorageMock struct {
	SaveUserFunc         func(ctx context.Context, user User) error
	UserFunc             func(ctx context.Context, etsyUserID int64) (User, error)
	TokenDetailsFunc     func(ctx context.Context, id int64) (TokenDetails, error)
	SaveTokenDetailsFunc func(ctx context.Context, td TokenDetails) error
}

func (s *StorageMock) SaveUser(ctx context.Context, user User) error {
	return s.SaveUserFunc(ctx, user)
}

func (s *StorageMock) User(ctx context.Context, etsyUserID int64) (User, error) {
	return s.UserFunc(ctx, etsyUserID)
}

func (s *StorageMock) TokenDetails(ctx context.Context, id int64) (TokenDetails, error) {
	return s.TokenDetailsFunc(ctx, id)
}

func (s *StorageMock) SaveTokenDetails(ctx context.Context, td TokenDetails) error {
	return s.SaveTokenDetailsFunc(ctx, td)
}
