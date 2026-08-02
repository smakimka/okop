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

// GetLocalizedNameForTagsRequest resolves names for specific tag ids. Unlike
// GetTagList it is not limited to the tags Steam currently publishes, so it can
// name tags that still appear on games after being dropped from the store
// vocabulary.
type GetLocalizedNameForTagsRequest struct {
	Language string `json:"language"`
	TagIDs   []int  `json:"tagids"`
}

type GetLocalizedNameForTagsResponse struct {
	Response struct {
		Tags []LocalizedTagInfo `json:"tags"`
	} `json:"response"`
}

type LocalizedTagInfo struct {
	TagID          int    `json:"tagid"`
	EnglishName    string `json:"english_name"`
	Name           string `json:"name"`
	NormalizedName string `json:"normalized_name"`
}
