package etsy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cooldarkdryplace/lowstock"

	"github.com/VictoriaMetrics/metrics"
	oauth "github.com/cooldarkdryplace/oauth1/etsy"
)

var (
	feedSuccessCounter = metrics.NewCounter(`etsy_feeds_api_calls{status="success"}`)
	feedFailureCounter = metrics.NewCounter(`etsy_feeds_api_calls{status="failure"}`)

	apiSuccessCounter = metrics.NewCounter(`etsy_open_api_calls{status="success"}`)
	apiFailureCounter = metrics.NewCounter(`etsy_open_api_calls{status="failure"}`)
)

type Etsy interface {
	Login(ctx context.Context) (string, oauth.TokenDetails, error)
	Callback(ctx context.Context, pin, token, secret string) (oauth.TokenDetails, error)
	HTTPClient(token, secret string) *http.Client
}

type EtsyClient struct {
	liveFeedsKey string
	etsy         Etsy
}

func NewClient(e Etsy, key string) *EtsyClient {
	return &EtsyClient{
		etsy:         e,
		liveFeedsKey: key,
	}
}

func (e *EtsyClient) Login(ctx context.Context, userID int64) (string, lowstock.TokenDetails, error) {
	loginURL, details, err := e.etsy.Login(ctx)
	if err != nil {
		return "", lowstock.TokenDetails{}, err
	}

	ltd := lowstock.TokenDetails{
		ID:          userID,
		Token:       details.Token,
		TokenSecret: details.TokenSecret,
	}

	return loginURL, ltd, nil
}

func (e *EtsyClient) Callback(ctx context.Context, pin, token, secret string) (lowstock.TokenDetails, error) {
	details, err := e.etsy.Callback(ctx, pin, token, secret)
	if err != nil {
		return lowstock.TokenDetails{}, err
	}

	ltd := lowstock.TokenDetails{
		Token:       details.Token,
		TokenSecret: details.TokenSecret,
	}

	return ltd, nil
}

func (e *EtsyClient) HTTPClient(accessToken, accessSecret string) *http.Client {
	return e.etsy.HTTPClient(accessToken, accessSecret)
}

type UserInfo struct {
	ID        int64  `json:"user_id"`
	LoginName string `json:"login_name"`
}

type UserInfoResponse struct {
	Count   int        `json:"count"`
	Results []UserInfo `json:"results"`
	Type    string     `json:"type"`
}

func (e *EtsyClient) UserID(accessToken, accessSecret string) (int64, error) {
	httpClient := e.HTTPClient(accessToken, accessSecret)

	uri := "https://openapi.etsy.com/v2/users/__SELF__"

	resp, err := httpClient.Get(uri)
	if err != nil {
		apiFailureCounter.Inc()
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		apiFailureCounter.Inc()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0, fmt.Errorf("failed to read response body: %w", err)
		}

		return 0, fmt.Errorf("bad response: %s, body: %s", resp.Status, string(body))
	}

	var infoResponse UserInfoResponse

	if err := json.NewDecoder(resp.Body).Decode(&infoResponse); err != nil {
		apiFailureCounter.Inc()
		return 0, err
	}

	apiSuccessCounter.Inc()

	if len(infoResponse.Results) < 1 {
		return 0, errors.New("no user info")
	}

	return infoResponse.Results[0].ID, nil
}

type shop struct {
	Name string `json:"shop_name"`
}

type listingInfo struct {
	ListingID       int64    `json:"listing_id"`
	State           string   `json:"state"`
	UserID          int64    `json:"user_id"`
	Quantity        int64    `json:"quantity"`
	Title           string   `json:"title"`
	SKU             []string `json:"sku"`
	CreationTSZ     int64    `json:"creation_tsz"`
	LastModifiedTSZ int64    `json:"last_modified_tsz"`
	Shop            shop     `json:"Shop"`
}

type pagination struct {
	EffectiveLimit  int `json:"effective_limit"`
	EffectiveOffset int `json:"effective_offset"`
	EffectivePage   int `json:"effective_page"`
	NextOffset      int `json:"next_offset"`
	NextPage        int `json:"next_page"`
}

type listingsResponse struct {
	Count      int           `json:"count"`
	Results    []listingInfo `json:"results"`
	Type       string        `json:"type"`
	Pagination pagination    `json:"pagination"`
}

func toLowstockUpdate(l listingInfo) lowstock.Update {
	return lowstock.Update{
		State:           l.State,
		Title:           l.Title,
		ShopName:        l.Shop.Name,
		ListingSKUs:     l.SKU,
		ListingID:       l.ListingID,
		UserID:          l.UserID,
		Quantity:        l.Quantity,
		CreationTSZ:     l.CreationTSZ,
		LastModifiedTSZ: l.LastModifiedTSZ,
	}
}

func toLowstockUpdates(listings []listingInfo) []lowstock.Update {
	updates := make([]lowstock.Update, 0, len(listings))

	for _, l := range listings {
		updates = append(updates, toLowstockUpdate(l))
	}

	return updates
}

var (
	limit     = "100"
	offset    = "0"
	timeLimit = "10"

	// TODO: use value from DB.
	lastUpdateTSZ = strconv.FormatInt(time.Now().Unix(), 10)
)

func (e *EtsyClient) Updates(ctx context.Context) ([]lowstock.Update, error) {
	timeOffset := lastUpdateTSZ

	params := url.Values{}
	params.Set("api_key", e.liveFeedsKey)
	params.Set("limit", limit)
	params.Set("offset", offset)
	params.Set("time_limit", timeLimit)
	params.Set("time_offset", timeOffset)

	resp, err := http.DefaultClient.Get("https://api.etsy.com/v2/feeds/listings/latest?" + params.Encode())
	if err != nil {
		feedFailureCounter.Inc()
		return nil, fmt.Errorf("failed to perform call to Etsy feeds: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		feedFailureCounter.Inc()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		return nil, fmt.Errorf("bad response: %s, body: %s", resp.Status, string(body))
	}

	var lResp = listingsResponse{}

	if err := json.NewDecoder(resp.Body).Decode(&lResp); err != nil {
		feedFailureCounter.Inc()
		return nil, fmt.Errorf("failed to decode feeds response: %w", err)
	}

	feedSuccessCounter.Inc()

	updates := toLowstockUpdates(lResp.Results)
	// TODO: update in DB
	lastUpdateTSZ = strconv.FormatInt(time.Now().Unix(), 10)

	return updates, nil
}
