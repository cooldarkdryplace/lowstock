package lowstock

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

const fallbackTimeout = 20 * time.Second

const (
	active      = "active"
	soldOut     = "sold_out"
	expired     = "expired"
	removed     = "removed"
	edit        = "edit"
	vacation    = "vacation"
	private     = "private"
	unavailable = "unavailable"
)

var ErrNotFound = errors.New("not found")

type TokenDetails struct {
	ID          int64
	Token       string
	TokenSecret string
}

type User struct {
	EtsyUserID  int64
	ChatUserID  int64
	ChatID      int64
	Token       string
	TokenSecret string
}

type Etsy interface {
	Callback(ctx context.Context, pin, token, secret string) (TokenDetails, error)
	Login(ctx context.Context, id int64) (string, TokenDetails, error)
	UserID(accessToken, accessSecret string) (int64, error)
	Updates(ctx context.Context) ([]Update, error)
}

type Storage interface {
	SaveUser(ctx context.Context, user User) error
	User(ctx context.Context, etsyUserID int64) (User, error)
	TokenDetails(ctx context.Context, id int64) (TokenDetails, error)
	SaveTokenDetails(ctx context.Context, td TokenDetails) error
}

type MessengerUpdate struct {
	ID      int64
	ChatID  int64
	UserID  int64
	Command string
	Text    string
}

type Messenger interface {
	SendLoginURL(text, url string, chatID int64) error
	SendTextMessage(msg string, chatID int64) error
	Updates(lastMsgID int64) ([]MessengerUpdate, error)
}

type LowStock struct {
	etsy      Etsy
	messenger Messenger
	storage   Storage

	mu           sync.Mutex
	lastUpdateID int64
}

func New(e Etsy, m Messenger, s Storage) *LowStock {
	ls := &LowStock{
		etsy:      e,
		messenger: m,
		storage:   s,
	}

	return ls
}

type Update struct {
	State           string
	Title           string
	ShopName        string
	ListingSKUs     []string
	ListingID       int64
	UserID          int64
	Quantity        int64
	CreationTSZ     int64
	LastModifiedTSZ int64
}

func (ls *LowStock) Updates(ctx context.Context) ([]Update, error) {
	updates, err := ls.etsy.Updates(ctx)
	if err != nil {
		log.Printf("Failed to get listing updates from Etsy: %s", err)
		return nil, err
	}

	return updates, nil
}

func (ls *LowStock) HandleEtsyUpdate(ctx context.Context, update Update) error {
	name := fmt.Sprintf(`etsy_updates_total{state=%q}`, update.State)
	metrics.GetOrCreateCounter(name).Inc()

	switch update.State {
	case soldOut:
		// Cache this data, I do not want a lot of requests here.
		user, err := ls.storage.User(ctx, update.UserID)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return fmt.Errorf("failed to get User record: %w", err)
			}
			return nil
		}

		msg := fmt.Sprintf("Low stock for SKU: %v, shop: %s", update.ListingSKUs, update.ShopName)
		if err := ls.messenger.SendTextMessage(msg, user.ChatID); err != nil {
			return fmt.Errorf("failed to send message via messenger: %w", err)
		}
	default:
		// noop
	}

	return nil
}

func (ls *LowStock) DoPin(ctx context.Context, msgUpdate MessengerUpdate) error {
	log.Printf("PIN: %#v", msgUpdate)
	details, err := ls.storage.TokenDetails(ctx, msgUpdate.UserID)
	if err != nil {
		return err
	}

	pin := strings.TrimSpace(strings.TrimPrefix(msgUpdate.Text, "/pin"))

	details, err = ls.etsy.Callback(ctx, pin, details.Token, details.TokenSecret)
	if err != nil {
		return fmt.Errorf("failed to handle Etsy OAuth callback: %w", err)
	}

	etsyUserID, err := ls.etsy.UserID(details.Token, details.TokenSecret)
	if err != nil {
		return err
	}

	user := User{
		EtsyUserID:  etsyUserID,
		ChatID:      msgUpdate.ChatID,
		ChatUserID:  msgUpdate.UserID,
		Token:       details.Token,
		TokenSecret: details.TokenSecret,
	}

	log.Printf("Saving user: %#v", user)

	if err := ls.storage.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("Failed to save user details: %w", err)
	}

	msg := "Success! You will be notified when products are sold out."
	if err := ls.messenger.SendTextMessage(msg, msgUpdate.ChatID); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}

func (ls *LowStock) DoStart(ctx context.Context, msgUpdate MessengerUpdate) error {
	uri, details, err := ls.etsy.Login(ctx, msgUpdate.UserID)
	if err != nil {
		return err
	}

	if err := ls.messenger.SendLoginURL("Disclaimer...", uri, msgUpdate.ChatID); err != nil {
		return err
	}

	if err := ls.storage.SaveTokenDetails(ctx, details); err != nil {
		return fmt.Errorf("failed to save oauth token details: %w", err)
	}

	return nil
}

func (ls *LowStock) DoHelp(ctx context.Context, msgUpdate MessengerUpdate) error {
	msg := "Supported commands: /start /pin /help"

	if err := ls.messenger.SendTextMessage(msg, msgUpdate.ChatID); err != nil {
		fmt.Errorf("failed to send help instructions: %w", err)
	}

	return nil
}

func (ls *LowStock) handleUpdate(ctx context.Context, msgUpdate MessengerUpdate) error {
	command := msgUpdate.Command
	ls.trackLastUpdateID(msgUpdate.ID)

	userIDStr := strconv.FormatInt(msgUpdate.UserID, 10)
	name := fmt.Sprintf(`tg_commands_total{command=%q, user_id=%q}`, command, userIDStr)
	metrics.GetOrCreateCounter(name).Inc()

	switch command {
	case "/pin":
		return ls.DoPin(ctx, msgUpdate)
	case "/start":
		return ls.DoStart(ctx, msgUpdate)
	case "/help":
		return ls.DoHelp(ctx, msgUpdate)
	default:
		log.Printf("Unsupported command: %s", command)
		return nil
	}

	return nil
}

func (ls *LowStock) handleUpdates(ctx context.Context, msgUpdates []MessengerUpdate) {
	for _, upd := range msgUpdates {
		if err := ls.handleUpdate(ctx, upd); err != nil {
			log.Printf("Failed to handle messenger update: %s", err)
		}
	}
}

func (ls *LowStock) trackLastUpdateID(ID int64) {
	ls.mu.Lock()
	if ls.lastUpdateID < ID {
		ls.lastUpdateID = ID
	}
	ls.mu.Unlock()
}

// ListenAndServe gets updates and processes them.
func (ls *LowStock) ListenAndServe(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msgUpdates, err := ls.messenger.Updates(ls.lastUpdateID + 1)
			if err != nil {
				log.Printf("Failed getting messenger msgUpdates: %s", err)
				time.Sleep(fallbackTimeout)
			}
			ls.handleUpdates(ctx, msgUpdates)
		}
	}
}
