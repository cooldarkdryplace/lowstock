package lowstock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestUpdates(t *testing.T) {
	etsyUpdates := []Update{
		Update{State: soldOut},
		Update{State: expired},
	}

	etsy := &EtsyMock{
		UpdatesFunc: func(ctx context.Context) ([]Update, error) {
			return etsyUpdates, nil
		},
	}

	ls := New(etsy, nil, nil)

	actualUpdates, err := ls.Updates(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if diff := cmp.Diff(actualUpdates, etsyUpdates); diff != "" {
		t.Errorf("Updates dot not match:\n%s", diff)
	}
}

func TestHandleEtsyUpdate(t *testing.T) {
	var (
		expectedChatID     int64 = 100500
		expectedUserID     int64 = 9500
		expectedEtsyUserID int64 = 5432
		expectedToken            = "test_token"
		expectedSecret           = "test_secret"
	)

	storage := &StorageMock{
		UserFunc: func(ctx context.Context, etsyUserID int64) (User, error) {
			return User{
				EtsyUserID:  expectedEtsyUserID,
				ChatUserID:  expectedUserID,
				ChatID:      expectedChatID,
				Token:       expectedToken,
				TokenSecret: expectedSecret,
			}, nil
		},
	}

	etsy := &EtsyMock{
		ListingSKUsFunc: func(ctx context.Context, id int64, accessToken, accessSecret string) ([]string, error) {
			return []string{"TestSKU#1", "TestSKU#2"}, nil
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

	ls := New(etsy, messenger, storage)

	update := Update{
		State:           soldOut,
		Title:           "Test product",
		ShopName:        "Test shop",
		ListingID:       42,
		UserID:          123456,
		Quantity:        0,
		CreationTSZ:     time.Now().Add(-24 * time.Hour).Unix(),
		LastModifiedTSZ: time.Now().Unix(),
	}

	if err := ls.HandleEtsyUpdate(context.Background(), update); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

func TestHandleEtsyUpdateUnsupportedState(t *testing.T) {
	states := []string{active, expired, removed, edit, vacation, private, unavailable}

	var (
		storage   = &StorageMock{}
		messenger = &MessengerMock{}
		etsy      = &EtsyMock{}
	)

	ls := New(etsy, messenger, storage)

	for _, state := range states {
		t.Run(state, func(t *testing.T) {
			update := Update{State: state}
			if err := ls.HandleEtsyUpdate(context.Background(), update); err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
		})
	}
}

func TestHandleEtsyUpdateUnknownUserID(t *testing.T) {
	storage := &StorageMock{
		UserFunc: func(ctx context.Context, etsyUserID int64) (User, error) {
			return User{}, ErrNotFound
		},
	}

	messenger := &MessengerMock{}
	etsy := &EtsyMock{}

	ls := New(etsy, messenger, storage)

	update := Update{
		State:           soldOut,
		Title:           "Test product",
		ShopName:        "Test shop",
		ListingID:       42,
		UserID:          123456,
		Quantity:        0,
		CreationTSZ:     time.Now().Add(-24 * time.Hour).Unix(),
		LastModifiedTSZ: time.Now().Unix(),
	}

	if err := ls.HandleEtsyUpdate(context.Background(), update); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

func TestDoEmptyPin(t *testing.T) {
	var (
		expectedChatID int64 = 100500
		expectedUserID int64 = 9500
		expectedError        = ErrEmptyPin
	)

	storage := &StorageMock{}
	etsy := &EtsyMock{}

	messageSent := false
	messenger := &MessengerMock{
		SendTextMessageFunc: func(msg string, chatID int64) error {
			if msg != emptyPinMsg {
				t.Error("Unexpected message")
			}

			if chatID != expectedChatID {
				t.Errorf("Got chat ID: %d, expected: %d", chatID, expectedChatID)
			}

			messageSent = true
			return nil
		},
	}

	ls := New(etsy, messenger, storage)

	update := MessengerUpdate{
		Command: "/pin",
		Text:    "/pin ",
		ChatID:  expectedChatID,
		UserID:  expectedUserID,
	}

	err := ls.DoPin(context.Background(), update)
	if !errors.Is(err, expectedError) {
		t.Errorf("Got error: %v, expected error: %s", err, expectedError)
	}

	if !messageSent {
		t.Error("Response message was not sent")
	}
}

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
			if msg != successMsg {
				t.Error("Unexpected message")
			}

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

func TestDoHelp(t *testing.T) {
	var expectedChatID int64 = 42

	etsy := &EtsyMock{}
	storage := &StorageMock{}

	messageSent := false

	messenger := &MessengerMock{
		SendTextMessageFunc: func(msg string, chatID int64) error {
			if msg != helpMsg {
				t.Error("Unexpected message")
			}

			if chatID != expectedChatID {
				t.Errorf("Got chat ID: %d, expected: %d", chatID, expectedChatID)
			}

			messageSent = true

			return nil
		},
	}

	ls := New(etsy, messenger, storage)

	update := MessengerUpdate{ChatID: expectedChatID}

	if err := ls.DoHelp(context.Background(), update); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if !messageSent {
		t.Error("Help message was not sent")
	}
}

func TestDoStart(t *testing.T) {
	var (
		expectedURL           = "https://example.com/login"
		expectedChatID  int64 = 42
		expectedDetails       = TokenDetails{
			Token:       "test_token",
			TokenSecret: "test_secret",
		}
	)

	storage := &StorageMock{
		SaveTokenDetailsFunc: func(ctx context.Context, td TokenDetails) error {
			if diff := cmp.Diff(expectedDetails, td); diff != "" {
				t.Errorf("Different TokenDetails:\n%s", diff)
			}

			return nil
		},
	}

	loginURLSent := false
	messenger := &MessengerMock{
		SendLoginURLFunc: func(msg, url string, chatID int64) error {
			loginURLSent = true

			if msg != startMsg {
				t.Error("Unexpected message")
			}

			if chatID != expectedChatID {
				t.Errorf("Got chat ID: %d, expected: %d", chatID, expectedChatID)
			}

			if url != expectedURL {
				t.Errorf("Got login URL: %s, expected: %s", url, expectedURL)
			}

			return nil
		},
	}

	etsy := &EtsyMock{
		LoginFunc: func(ctx context.Context, id int64) (string, TokenDetails, error) {
			return expectedURL, expectedDetails, nil
		},
	}

	ls := New(etsy, messenger, storage)

	update := MessengerUpdate{ChatID: expectedChatID}

	if err := ls.DoStart(context.Background(), update); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if !loginURLSent {
		t.Error("Login URL was not sent")
	}
}

type EtsyMock struct {
	CallbackFunc    func(ctx context.Context, pin, token, secret string) (TokenDetails, error)
	LoginFunc       func(ctx context.Context, id int64) (string, TokenDetails, error)
	ListingSKUsFunc func(ctx context.Context, id int64, accessToken, accessSecret string) ([]string, error)
	UserIDFunc      func(accessToken, accessSecret string) (int64, error)
	UpdatesFunc     func(ctx context.Context) ([]Update, error)
}

func (e *EtsyMock) Callback(ctx context.Context, pin, token, secret string) (TokenDetails, error) {
	return e.CallbackFunc(ctx, pin, token, secret)
}

func (e *EtsyMock) Login(ctx context.Context, id int64) (string, TokenDetails, error) {
	return e.LoginFunc(ctx, id)
}

func (e *EtsyMock) ListingSKUs(ctx context.Context, id int64, accessToken, accessSecret string) ([]string, error) {
	return e.ListingSKUsFunc(ctx, id, accessToken, accessSecret)
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
