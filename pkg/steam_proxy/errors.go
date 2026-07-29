package steam_proxy

import "errors"

var (
	ErrRateLimited = errors.New("rate limit hit")
)
