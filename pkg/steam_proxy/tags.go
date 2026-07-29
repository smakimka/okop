package steam_proxy

type GetTagListRequest struct {
	Language        string `json:"language"`
	HaveVersionHash string `json:"have_version_hash,omitempty"`
}

type GetTagListResponse struct {
	Response struct {
		VersionHash string    `json:"version_hash"`
		Tags        []TagInfo `json:"tags"`
	} `json:"response"`
}

type TagInfo struct {
	TagID int    `json:"tagid"`
	Name  string `json:"name"`
}
