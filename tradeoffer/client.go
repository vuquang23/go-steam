package tradeoffer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/vuquang23/go-steam/community"
	"github.com/vuquang23/go-steam/economy/inventory"
	"github.com/vuquang23/go-steam/netutil"
	"github.com/vuquang23/go-steam/steamid"
)

type APIKey string

const apiUrl = "https://api.steampowered.com/IEconService/%s/v%d"

const defaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36"

type Client struct {
	client    *http.Client
	key       APIKey
	sessionId string
}

func NewClient(key APIKey, sessionId string) *Client {
	c := &Client{
		new(http.Client),
		key,
		sessionId,
	}
	return c
}

func (c *Client) SetProxy(proxy string) error {
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		return err
	}
	c.client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
	return nil
}

func (c *Client) SetCookies(cookies []*http.Cookie) error {
	return community.SetCookies(c.client, cookies)
}

func (c *Client) GetOffer(offerId uint64) (*TradeOfferResult, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(apiUrl, "GetTradeOffer", 1)+"?"+netutil.ToUrlValues(map[string]string{
		"key":          string(c.key),
		"tradeofferid": strconv.FormatUint(offerId, 10),
		"language":     "en_us",
	}).Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	t := new(struct {
		Response *TradeOfferResult
	})
	if err = json.NewDecoder(resp.Body).Decode(t); err != nil {
		return nil, err
	}
	if t.Response == nil || t.Response.Offer == nil {
		return nil, newSteamErrorf("steam returned empty offer result\n")
	}

	return t.Response, nil
}

func (c *Client) GetOffers(getSent bool, getReceived bool, getDescriptions bool, activeOnly bool, historicalOnly bool, timeHistoricalCutoff *uint32) (*TradeOffersResult, error) {
	if !getSent && !getReceived {
		return nil, errors.New("getSent and getReceived can't be both false")
	}

	params := map[string]string{
		"key": string(c.key),
	}
	if getSent {
		params["get_sent_offers"] = "1"
	}
	if getReceived {
		params["get_received_offers"] = "1"
	}
	if getDescriptions {
		params["get_descriptions"] = "1"
		params["language"] = "en_us"
	}
	if activeOnly {
		params["active_only"] = "1"
	}
	if historicalOnly {
		params["historical_only"] = "1"
	}
	if timeHistoricalCutoff != nil {
		params["time_historical_cutoff"] = strconv.FormatUint(uint64(*timeHistoricalCutoff), 10)
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(apiUrl, "GetTradeOffers", 1)+"?"+netutil.ToUrlValues(params).Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	t := new(struct {
		Response *TradeOffersResult
	})
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(bodyBytes, t); err != nil {
		return nil, fmt.Errorf("unmarshal error: %s, response: %s", err, bodyBytes)
	}
	if t.Response == nil {
		return nil, newSteamErrorf("steam returned empty offers result\n")
	}
	return t.Response, nil
}

// action() is used by Decline() and Cancel()
// Steam only return success and error fields for malformed requests,
// hence client shall use GetOffer() to check action result
// It is also possible to implement Decline/Cancel using steamcommunity,
// which have more predictable responses
func (c *Client) action(method string, version uint, offerId uint64) error {
	req := netutil.NewPostForm(fmt.Sprintf(apiUrl, method, version), netutil.ToUrlValues(map[string]string{
		"key":          string(c.key),
		"tradeofferid": strconv.FormatUint(offerId, 10),
	}))
	req.Header.Set("User-Agent", defaultUserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf(method+" error: status code %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) Decline(offerId uint64) error {
	return c.action("DeclineTradeOffer", 1, offerId)
}

func (c *Client) Cancel(offerId uint64) error {
	return c.action("CancelTradeOffer", 1, offerId)
}

// Accept received trade offer
// It is best to confirm that offer was actually accepted
// by calling GetOffer after Accept and checking offer state
func (c *Client) Accept(offerId uint64) error {
	baseurl := fmt.Sprintf("https://steamcommunity.com/tradeoffer/%d/", offerId)
	req := netutil.NewPostForm(baseurl+"accept", netutil.ToUrlValues(map[string]string{
		"sessionid":    c.sessionId,
		"serverid":     "1",
		"tradeofferid": strconv.FormatUint(offerId, 10),
	}))
	req.Header.Add("Referer", baseurl)
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	t := new(struct {
		StrError string `json:"strError"`
	})
	if err = json.NewDecoder(resp.Body).Decode(t); err != nil {
		return err
	}
	if t.StrError != "" {
		return newSteamErrorf("accept error: %v\n", t.StrError)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("accept error: status code %d", resp.StatusCode)
	}
	return nil
}

type TradeItem struct {
	AppId      uint32 `json:"appid"`
	ContextId  uint64 `json:"contextid,string"`
	Amount     uint64 `json:"amount"`
	AssetId    uint64 `json:"assetid,string,omitempty"`
	CurrencyId uint64 `json:"currencyid,string,omitempty"`
}

// Sends a new trade offer to the given Steam user. You can optionally specify an access token if you've got one.
// In addition, `counteredOfferId` can be non-nil, indicating the trade offer this is a counter for.
// On success returns trade offer id
func (c *Client) Create(other steamid.SteamId, accessToken *string, myItems, theirItems []TradeItem, counteredOfferId *uint64, message string) (uint64, error) {
	// Create new trade offer status
	to := map[string]interface{}{
		"newversion": true,
		"version":    3,
		"me": map[string]interface{}{
			"assets":   myItems,
			"currency": make([]struct{}, 0),
			"ready":    false,
		},
		"them": map[string]interface{}{
			"assets":   theirItems,
			"currency": make([]struct{}, 0),
			"ready":    false,
		},
	}

	jto, err := json.Marshal(to)
	if err != nil {
		panic(err)
	}

	// Create url parameters for request
	data := map[string]string{
		"sessionid":         c.sessionId,
		"serverid":          "1",
		"partner":           other.ToString(),
		"tradeoffermessage": message,
		"json_tradeoffer":   string(jto),
	}

	var referer string
	if counteredOfferId != nil {
		referer = fmt.Sprintf("https://steamcommunity.com/tradeoffer/%d/", *counteredOfferId)
		data["tradeofferid_countered"] = strconv.FormatUint(*counteredOfferId, 10)
	} else {
		// Add token for non-friend offers
		if accessToken != nil {
			params := map[string]string{
				"trade_offer_access_token": *accessToken,
			}
			paramsJson, err := json.Marshal(params)
			if err != nil {
				panic(err)
			}

			data["trade_offer_create_params"] = string(paramsJson)

			referer = "https://steamcommunity.com/tradeoffer/new/?partner=" + strconv.FormatUint(uint64(other.GetAccountId()), 10) + "&token=" + *accessToken
		} else {

			referer = "https://steamcommunity.com/tradeoffer/new/?partner=" + strconv.FormatUint(uint64(other.GetAccountId()), 10)
		}
	}

	// Create request
	req := netutil.NewPostForm("https://steamcommunity.com/tradeoffer/new/send", netutil.ToUrlValues(data))
	req.Header.Add("Referer", referer)
	req.Header.Set("User-Agent", defaultUserAgent)

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	t := new(struct {
		StrError     string `json:"strError"`
		TradeOfferId uint64 `json:"tradeofferid,string"`
	})
	if err = json.NewDecoder(resp.Body).Decode(t); err != nil {
		return 0, err
	}
	// strError code descriptions:
	// 15	invalide trade access token
	// 16	timeout
	// 20	wrong contextid
	// 25	can't send more offers until some is accepted/cancelled...
	// 26	object is not in our inventory
	// error code names are in internal/steamlang/enums.go EResult_name
	if t.StrError != "" {
		return 0, newSteamErrorf("create error: %v\n", t.StrError)
	}
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("create error: status code %d", resp.StatusCode)
	}
	if t.TradeOfferId == 0 {
		return 0, newSteamErrorf("create error: steam returned 0 for trade offer id")
	}
	return t.TradeOfferId, nil
}

func (c *Client) GetOwnInventory(contextId uint64, appId uint32, tradableOnly bool) (*inventory.Inventory, error) {
	return inventory.GetOwnInventory(c.client, contextId, appId, tradableOnly)
}

func (c *Client) GetPartnerInventory(other steamid.SteamId, contextId uint64, appId uint32, offerId *uint64) (*inventory.Inventory, error) {
	return inventory.GetFullInventory(func() (*inventory.PartialInventory, error) {
		return c.getPartialPartnerInventory(other, contextId, appId, offerId, nil)
	}, func(start uint) (*inventory.PartialInventory, error) {
		return c.getPartialPartnerInventory(other, contextId, appId, offerId, &start)
	})
}

func (c *Client) getPartialPartnerInventory(other steamid.SteamId, contextId uint64, appId uint32, offerId *uint64, start *uint) (*inventory.PartialInventory, error) {
	data := map[string]string{
		"sessionid": c.sessionId,
		"partner":   other.ToString(),
		"contextid": strconv.FormatUint(contextId, 10),
		"appid":     strconv.FormatUint(uint64(appId), 10),
	}
	if start != nil {
		data["start"] = strconv.FormatUint(uint64(*start), 10)
	}

	baseUrl := "https://steamcommunity.com/tradeoffer/%v/"
	if offerId != nil {
		baseUrl = fmt.Sprintf(baseUrl, *offerId)
	} else {
		baseUrl = fmt.Sprintf(baseUrl, "new")
	}

	req, err := http.NewRequest("GET", baseUrl+"partnerinventory/?"+netutil.ToUrlValues(data).Encode(), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Referer", baseUrl+"?partner="+strconv.FormatUint(uint64(other.GetAccountId()), 10))
	req.Header.Set("User-Agent", defaultUserAgent)

	return inventory.DoInventoryRequest(c.client, req)
}

// Can be used to verify accepted tradeoffer and find out received asset ids
func (c *Client) GetTradeReceipt(tradeId uint64) ([]*TradeReceiptItem, error) {
	url := fmt.Sprintf("https://steamcommunity.com/trade/%d/receipt", tradeId)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	items, err := parseTradeReceipt(respBody)
	if err != nil {
		return nil, newSteamErrorf("failed to parse trade receipt: %v", err)
	}
	return items, nil
}

// Get duration of escrow in days. Call this before sending a trade offer
func (c *Client) GetPartnerEscrowDuration(other steamid.SteamId, accessToken *string) (*EscrowDuration, error) {
	data := map[string]string{
		"partner": strconv.FormatUint(uint64(other.GetAccountId()), 10),
	}
	if accessToken != nil {
		data["token"] = *accessToken
	}
	return c.getEscrowDuration("https://steamcommunity.com/tradeoffer/new/?" + netutil.ToUrlValues(data).Encode())
}

// Get duration of escrow in days. Call this after receiving a trade offer
func (c *Client) GetOfferEscrowDuration(offerId uint64) (*EscrowDuration, error) {
	return c.getEscrowDuration("http://steamcommunity.com/tradeoffer/" + strconv.FormatUint(offerId, 10))
}

func (c *Client) getEscrowDuration(queryUrl string) (*EscrowDuration, error) {
	req, err := http.NewRequest(http.MethodGet, queryUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve escrow duration: %v", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	escrowDuration, err := parseEscrowDuration(respBody)
	if err != nil {
		return nil, newSteamErrorf("failed to parse escrow duration: %v", err)
	}
	return escrowDuration, nil
}

func (c *Client) GetOfferWithRetry(offerId uint64, retryCount int, retryDelay time.Duration) (*TradeOfferResult, error) {
	var res *TradeOfferResult
	return res, withRetry(
		func() (err error) {
			res, err = c.GetOffer(offerId)
			return err
		}, retryCount, retryDelay)
}

func (c *Client) GetOffersWithRetry(getSent bool, getReceived bool, getDescriptions bool, activeOnly bool, historicalOnly bool, timeHistoricalCutoff *uint32, retryCount int, retryDelay time.Duration) (*TradeOffersResult, error) {
	var res *TradeOffersResult
	return res, withRetry(
		func() (err error) {
			res, err = c.GetOffers(getSent, getReceived, getDescriptions, activeOnly, historicalOnly, timeHistoricalCutoff)
			return err
		}, retryCount, retryDelay)
}

func (c *Client) DeclineWithRetry(offerId uint64, retryCount int, retryDelay time.Duration) error {
	return withRetry(
		func() error {
			return c.Decline(offerId)
		}, retryCount, retryDelay)
}

func (c *Client) CancelWithRetry(offerId uint64, retryCount int, retryDelay time.Duration) error {
	return withRetry(
		func() error {
			return c.Cancel(offerId)
		}, retryCount, retryDelay)
}

func (c *Client) AcceptWithRetry(offerId uint64, retryCount int, retryDelay time.Duration) error {
	return withRetry(
		func() error {
			return c.Accept(offerId)
		}, retryCount, retryDelay)
}

func (c *Client) CreateWithRetry(other steamid.SteamId, accessToken *string, myItems, theirItems []TradeItem, counteredOfferId *uint64, message string, retryCount int, retryDelay time.Duration) (uint64, error) {
	var res uint64
	return res, withRetry(
		func() (err error) {
			res, err = c.Create(other, accessToken, myItems, theirItems, counteredOfferId, message)
			return err
		}, retryCount, retryDelay)
}

func (c *Client) GetOwnInventoryWithRetry(contextId uint64, appId uint32, retryCount int, retryDelay time.Duration, tradableOnly bool) (*inventory.Inventory, error) {
	var res *inventory.Inventory
	return res, withRetry(
		func() (err error) {
			res, err = c.GetOwnInventory(contextId, appId, tradableOnly)
			return err
		}, retryCount, retryDelay)
}

func (c *Client) GetPartnerInventoryWithRetry(other steamid.SteamId, contextId uint64, appId uint32, offerId *uint64, retryCount int, retryDelay time.Duration) (*inventory.Inventory, error) {
	var res *inventory.Inventory
	return res, withRetry(
		func() (err error) {
			res, err = c.GetPartnerInventory(other, contextId, appId, offerId)
			return err
		}, retryCount, retryDelay)
}

func (c *Client) GetTradeReceiptWithRetry(tradeId uint64, retryCount int, retryDelay time.Duration) ([]*TradeReceiptItem, error) {
	var res []*TradeReceiptItem
	return res, withRetry(
		func() (err error) {
			res, err = c.GetTradeReceipt(tradeId)
			return err
		}, retryCount, retryDelay)
}

func withRetry(f func() error, retryCount int, retryDelay time.Duration) error {
	if retryCount <= 0 {
		panic("retry count must be more than 0")
	}
	i := 0
	for {
		i++
		if err := f(); err != nil {
			// If we got steam error do not retry
			if _, ok := err.(*SteamError); ok {
				return err
			}
			if i == retryCount {
				return err
			}
			time.Sleep(retryDelay)
			continue
		}
		break
	}
	return nil
}
