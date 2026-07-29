package steam_proxy

type App struct {
	AppID             int    `json:"appid"`
	Name              string `json:"name"`
	LastModified      int64  `json:"last_modified"`
	PriceChangeNumber int64  `json:"price_change_number"`
}

type AppListResponse struct {
	Response struct {
		Apps            []App  `json:"apps"`
		HaveMoreResults bool   `json:"have_more_results"`
		LastAppID       uint32 `json:"last_appid"`
	} `json:"response"`
}
