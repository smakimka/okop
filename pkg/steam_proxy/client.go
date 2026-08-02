package steam_proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type Client struct {
	instance *http.Client

	name     string
	proxyURL *url.URL
}

type Config struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func New(cfg Config) (*Client, error) {
	var proxyURL *url.URL

	if cfg.URL != "" {
		var err error
		proxyURL, err = url.Parse(cfg.URL)
		if err != nil {
			return nil, err
		}
	}

	c := &Client{
		instance: &http.Client{},

		name:     cfg.Name,
		proxyURL: proxyURL,
	}

	c.instance.Transport = &http.Transport{
		Proxy: c.proxy,
	}

	return c, nil
}

func (c *Client) proxy(*http.Request) (*url.URL, error) {
	return c.proxyURL, nil
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	rsp, err := c.instance.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *Client) doJSON(req *http.Request, dst any) error {
	body, err := c.do(req)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, dst); err != nil {
		return err
	}

	return nil
}

func (c *Client) GameInfos(ctx context.Context, ids []int) (*GetItemsResponse, error) {
	gameIDs := make([]StoreItemID, 0, len(ids))
	for _, id := range ids {
		gameIDs = append(gameIDs, StoreItemID{AppID: uint32(id)})
	}

	reqData := GetItemsRequest{
		IDs: gameIDs,
		Context: StoreContext{
			Language:    "english",
			CountryCode: "US",
			SteamRealm:  1,
		},
		DataRequest: DataRequest{
			IncludeAssets:    true,
			IncludeRelease:   true,
			IncludePlatforms: true,
			IncludeBasicInfo: true,
			IncludeTagCount:  20,
			IncludeReviews:   true,
		},
	}

	rawReqData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	q := url.Values{}

	q.Add("input_json", string(rawReqData))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://api.steampowered.com/IStoreBrowseService/GetItems/v1/?"+q.Encode(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	var resp GetItemsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) AppsList(ctx context.Context, apiKey string, lastAppID int) (*AppListResponse, error) {
	q := url.Values{}

	q.Add("key", apiKey)
	q.Add("max_results", "50000")
	if lastAppID != 0 {
		q.Add("last_appid", strconv.Itoa(lastAppID))
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://api.steampowered.com/IStoreService/GetAppList/v1/?"+q.Encode(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	var rsp AppListResponse
	if err := c.doJSON(req, &rsp); err != nil {
		return nil, err
	}

	return &rsp, nil
}

func (c *Client) Tags(ctx context.Context) (*GetTagListResponse, error) {
	reqData := GetTagListRequest{
		Language: "english",
	}

	rawReqData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Add("input_json", string(rawReqData))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://api.steampowered.com/IStoreService/GetTagList/v1/?"+q.Encode(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	var resp GetTagListResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// TagNames resolves names for the given tag ids. Use it for tags Tags does not
// cover: Steam keeps returning ids on games after dropping them from the store
// tag list, and those can only be named by asking for them explicitly.
func (c *Client) TagNames(ctx context.Context, tagIDs []int) (*GetLocalizedNameForTagsResponse, error) {
	reqData := GetLocalizedNameForTagsRequest{
		Language: "english",
		TagIDs:   tagIDs,
	}

	rawReqData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Add("input_json", string(rawReqData))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://api.steampowered.com/IStoreService/GetLocalizedNameForTags/v1/?"+q.Encode(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	var resp GetLocalizedNameForTagsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) Name() string {
	return c.name
}
