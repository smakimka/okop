// Package steamstore contains request/response types for Steam's
// IStoreBrowseService/GetItems/v1 endpoint.
//
// This endpoint is undocumented by Valve. The shapes below are reverse-engineered
// from observed responses and Steam's public protobuf definitions; field availability
// depends on which include_* flags you set in DataRequest, and Valve may change
// things without notice.
//
// Endpoint:
//
//	GET https://api.steampowered.com/IStoreBrowseService/GetItems/v1/
//	    ?input_json=<url-encoded JSON of GetItemsRequest>
package steam_proxy

// ---------------------------------------------------------------------------
// Request
// ---------------------------------------------------------------------------

// GetItemsRequest is the top-level payload encoded into the input_json query param.
type GetItemsRequest struct {
	IDs         []StoreItemID `json:"ids"`
	Context     StoreContext  `json:"context"`
	DataRequest DataRequest   `json:"data_request"`
}

// StoreItemID identifies a single item to look up. Set exactly one of the
// three fields per entry. You can mix types within the same IDs slice.
type StoreItemID struct {
	AppID     uint32 `json:"appid,omitempty"`
	PackageID uint32 `json:"packageid,omitempty"`
	BundleID  uint32 `json:"bundleid,omitempty"`
}

// StoreContext drives localization, regional pricing, and content filtering.
type StoreContext struct {
	// Steam language code, e.g. "english", "french", "schinese", "brazilian".
	Language string `json:"language"`
	// ISO 3166-1 alpha-2 country code, e.g. "US", "DE", "JP".
	CountryCode string `json:"country_code"`
	// 1 = global Steam, 2 = Steam China (Perfect World). Almost always 1.
	SteamRealm int32 `json:"steam_realm"`
}

// DataRequest selects which sections of the response are populated.
// All fields are optional; omitted/false fields produce smaller responses.
type DataRequest struct {
	IncludeAssets                 bool `json:"include_assets,omitempty"`
	IncludeRelease                bool `json:"include_release,omitempty"`
	IncludePlatforms              bool `json:"include_platforms,omitempty"`
	IncludeAllPurchaseOptions     bool `json:"include_all_purchase_options,omitempty"`
	IncludeScreenshots            bool `json:"include_screenshots,omitempty"`
	IncludeTrailers               bool `json:"include_trailers,omitempty"`
	IncludeRatings                bool `json:"include_ratings,omitempty"`
	IncludeReviews                bool `json:"include_reviews,omitempty"`
	IncludeBasicInfo              bool `json:"include_basic_info,omitempty"`
	IncludeSupportedLanguages     bool `json:"include_supported_languages,omitempty"`
	IncludeFullDescription        bool `json:"include_full_description,omitempty"`
	IncludeIncludedItems          bool `json:"include_included_items,omitempty"`
	IncludeAssetsWithoutOverrides bool `json:"include_assets_without_overrides,omitempty"`
	ApplyUserFilters              bool `json:"apply_user_filters,omitempty"`
	// Number of top community tags to include (0 = none).
	IncludeTagCount int32 `json:"include_tag_count,omitempty"`
}

// ---------------------------------------------------------------------------
// Response
// ---------------------------------------------------------------------------

// GetItemsResponse is the top-level wrapper Steam returns.
type GetItemsResponse struct {
	Response GetItemsResponseBody `json:"response"`
}

// GetItemsResponseBody holds the per-item results in the same order requested.
type GetItemsResponseBody struct {
	StoreItems []StoreItem `json:"store_items"`
}

// StoreItem is a single entry in the response. Most fields are optional and
// only populated when the corresponding include_* flag was set.
type StoreItem struct {
	// Common identity fields - always returned on success.
	ItemType                        int32  `json:"item_type"` // 0=app, 1=package, 2=bundle
	ID                              uint32 `json:"id"`
	Success                         int32  `json:"success"` // 1 = found
	Visible                         bool   `json:"visible,omitempty"`
	UnvailableForCountryRestriction bool   `json:"unvailable_for_country_restriction,omitempty"` // sic - typo is Steam's

	Name          string `json:"name,omitempty"`
	StoreURLPath  string `json:"store_url_path,omitempty"`
	AppID         uint32 `json:"appid,omitempty"`
	Type          int32  `json:"type,omitempty"`
	IsFree        bool   `json:"is_free,omitempty"`
	IsEarlyAccess bool   `json:"is_early_access,omitempty"`

	ContentDescriptorIDs []int32 `json:"content_descriptorids,omitempty"`
	TagIDs               []int32 `json:"tagids,omitempty"`

	Categories  *Categories  `json:"categories,omitempty"`
	Reviews     *Reviews     `json:"reviews,omitempty"`
	BasicInfo   *BasicInfo   `json:"basic_info,omitempty"`
	Assets      *Assets      `json:"assets,omitempty"`
	Release     *Release     `json:"release,omitempty"`
	Platforms   *Platforms   `json:"platforms,omitempty"`
	Tags        []Tag        `json:"tags,omitempty"`
	Screenshots *Screenshots `json:"screenshots,omitempty"`
	Trailers    *Trailers    `json:"trailers,omitempty"`
	Ratings     []Rating     `json:"ratings,omitempty"`

	BestPurchaseOption *PurchaseOption  `json:"best_purchase_option,omitempty"`
	PurchaseOptions    []PurchaseOption `json:"purchase_options,omitempty"`
	IncludedItems      *IncludedItems   `json:"included_items,omitempty"`

	FullDescription    string              `json:"full_description,omitempty"`
	SupportedLanguages []SupportedLanguage `json:"supported_languages,omitempty"`

	RelatedItems *RelatedItems `json:"related_items,omitempty"`
}

// Categories lists feature flags (achievements, cloud saves, controller, etc.).
type Categories struct {
	SupportedPlayerCategoryIDs []int32 `json:"supported_player_category_ids,omitempty"`
	FeatureCategoryIDs         []int32 `json:"feature_category_ids,omitempty"`
	ControllerCategoryIDs      []int32 `json:"controller_category_ids,omitempty"`
}

// Reviews is the summary blob (positive %, total count, "Mostly Positive" string).
type Reviews struct {
	Summary                 *ReviewSummary `json:"summary_filtered,omitempty"`
	SummaryUnfiltered       *ReviewSummary `json:"summary_unfiltered,omitempty"`
	SummaryLanguageSpecific *ReviewSummary `json:"summary_language_specific,omitempty"`
}

// ReviewSummary describes a single review aggregate.
type ReviewSummary struct {
	ReviewCount      int32  `json:"review_count,omitempty"`
	PercentPositive  int32  `json:"percent_positive,omitempty"`
	ReviewScore      int32  `json:"review_score,omitempty"`
	ReviewScoreLabel string `json:"review_score_label,omitempty"` // e.g. "Overwhelmingly Positive"
}

// BasicInfo holds developer/publisher/franchise metadata and short copy.
type BasicInfo struct {
	ShortDescription   string              `json:"short_description,omitempty"`
	CapsuleHeadline    string              `json:"capsule_headline,omitempty"`
	Publishers         []Creator           `json:"publishers,omitempty"`
	Developers         []Creator           `json:"developers,omitempty"`
	Franchises         []Creator           `json:"franchises,omitempty"`
	SupportedLanguages []SupportedLanguage `json:"supported_languages,omitempty"`
	ContentDescriptors []ContentDescriptor `json:"content_descriptors,omitempty"`
	Website            string              `json:"website,omitempty"`
}

// Creator is a developer, publisher, or franchise reference.
type Creator struct {
	Name                 string `json:"name"`
	CreatorClanAccountID uint64 `json:"creator_clan_account_id,omitempty"`
}

// SupportedLanguage describes language support for an app.
type SupportedLanguage struct {
	ELanguage int32 `json:"elanguage,omitempty"`
	Supported bool  `json:"supported,omitempty"`
	FullAudio bool  `json:"full_audio,omitempty"`
	Subtitles bool  `json:"subtitles,omitempty"`
}

// ContentDescriptor is a mature-content warning entry.
type ContentDescriptor struct {
	ID          int32  `json:"id,omitempty"`
	Description string `json:"description,omitempty"`
}

// Assets contains the CDN paths for store/library imagery and trailers.
// To build a full URL, prepend AssetURLFormat (replacing ${FILENAME}) to each filename field.
type Assets struct {
	AssetURLFormat   string `json:"asset_url_format,omitempty"`
	MainCapsule      string `json:"main_capsule,omitempty"`
	SmallCapsule     string `json:"small_capsule,omitempty"`
	Header           string `json:"header,omitempty"`
	PackageHeader    string `json:"package_header,omitempty"`
	PageBackground   string `json:"page_background,omitempty"`
	HeroCapsule      string `json:"hero_capsule,omitempty"`
	HeroCapsule2x    string `json:"hero_capsule_2x,omitempty"`
	LibraryCapsule   string `json:"library_capsule,omitempty"`
	LibraryCapsule2x string `json:"library_capsule_2x,omitempty"`
	LibraryHero      string `json:"library_hero,omitempty"`
	LibraryHero2x    string `json:"library_hero_2x,omitempty"`
	LibraryLogo      string `json:"library_logo,omitempty"`
	CommunityIcon    string `json:"community_icon,omitempty"`
	Raw              string `json:"raw,omitempty"`
}

// Release captures release-date metadata.
type Release struct {
	SteamReleaseDate         int64  `json:"steam_release_date,omitempty"`    // unix timestamp
	OriginalReleaseDate      int64  `json:"original_release_date,omitempty"` // unix timestamp
	OriginalSteamReleaseDate int64  `json:"original_steam_release_date,omitempty"`
	IsComingSoon             bool   `json:"is_coming_soon,omitempty"`
	IsPreload                bool   `json:"is_preload,omitempty"`
	CustomReleaseDateMessage string `json:"custom_release_date_message,omitempty"`
	IsAbridgedReleaseDate    bool   `json:"is_abridged_release_date,omitempty"`
}

// Platforms describes OS support, Steam Deck compatibility, VR, and controllers.
type Platforms struct {
	SteamDeckCompatCategory int32      `json:"steam_deck_compat_category,omitempty"` // 0=unknown,1=unsupported,2=playable,3=verified
	VRSupport               *VRSupport `json:"vr_support,omitempty"`
	Windows                 bool       `json:"windows,omitempty"`
	Mac                     bool       `json:"mac,omitempty"`
	Linux                   bool       `json:"linux,omitempty"`
	SteamOSLinux            bool       `json:"steamos_linux,omitempty"`
	ControllerSupport       string     `json:"controller_support,omitempty"` // "none", "partial", "full"
}

// VRSupport flags which VR ecosystems are supported.
type VRSupport struct {
	VRSupportedHeadsets []int32 `json:"vrsupported_headsets,omitempty"`
	VRSupportedInput    []int32 `json:"vrsupported_input,omitempty"`
	VRSupportedPlayArea []int32 `json:"vrsupported_play_area,omitempty"`
}

// Tag is a community tag with vote weight.
type Tag struct {
	TagID  int32 `json:"tagid"`
	Weight int32 `json:"weight,omitempty"`
}

// Screenshots groups all the screenshot entries.
type Screenshots struct {
	AllAgesScreenshots       []Screenshot `json:"all_ages_screenshots,omitempty"`
	MatureContentScreenshots []Screenshot `json:"mature_content_screenshots,omitempty"`
}

// Screenshot is a single still image entry.
type Screenshot struct {
	Filename string `json:"filename,omitempty"`
	Ordinal  int32  `json:"ordinal,omitempty"`
}

// Trailers groups all the trailer/movie entries.
type Trailers struct {
	HighlightedTrailers []Trailer `json:"highlighted_trailers,omitempty"`
	OtherTrailers       []Trailer `json:"other_trailers,omitempty"`
}

// Trailer holds the available encoded video files plus a poster image.
type Trailer struct {
	TrailerName      string        `json:"trailer_name,omitempty"`
	TrailerURLFormat string        `json:"trailer_url_format,omitempty"`
	TrailerBaseID    uint64        `json:"trailer_base_id,omitempty"`
	TrailerCategory  int32         `json:"trailer_category,omitempty"`
	Trailer480p      []TrailerFile `json:"trailer_480p,omitempty"`
	TrailerMax       []TrailerFile `json:"trailer_max,omitempty"`
	ScreenshotMedium string        `json:"screenshot_medium,omitempty"`
	ScreenshotFull   string        `json:"screenshot_full,omitempty"`
}

// TrailerFile is one encoding (mp4, webm, ...) of a trailer.
type TrailerFile struct {
	Filename string `json:"filename,omitempty"`
	Type     string `json:"type,omitempty"` // "video/mp4", "video/webm"
}

// Rating is one age-rating-board entry (ESRB, PEGI, USK, ...).
type Rating struct {
	RatingBoard                string   `json:"rating_board,omitempty"`
	Rating                     string   `json:"rating,omitempty"`
	RatingGeneratedDescription string   `json:"rating_generated_description,omitempty"`
	Descriptors                []string `json:"descriptors,omitempty"`
	InteractiveElements        []string `json:"interactive_elements,omitempty"`
	UseAgeGate                 bool     `json:"use_age_gate,omitempty"`
	RequiredAge                int32    `json:"required_age,omitempty"`
	ImageURL                   string   `json:"image_url,omitempty"`
}

// PurchaseOption is one buyable SKU (a package or bundle) for the item.
type PurchaseOption struct {
	PackageID                 uint32           `json:"packageid,omitempty"`
	BundleID                  uint32           `json:"bundleid,omitempty"`
	PurchaseOptionName        string           `json:"purchase_option_name,omitempty"`
	FinalPriceInCents         int64            `json:"final_price_in_cents,string,omitempty"`
	OriginalPriceInCents      int64            `json:"original_price_in_cents,string,omitempty"`
	UserFinalPriceInCents     int64            `json:"user_final_price_in_cents,string,omitempty"`
	FormattedFinalPrice       string           `json:"formatted_final_price,omitempty"`
	FormattedOriginalPrice    string           `json:"formatted_original_price,omitempty"`
	DiscountPct               int32            `json:"discount_pct,omitempty"`
	UserDiscountPct           int32            `json:"user_discount_pct,omitempty"`
	BundleDiscountPct         int32            `json:"bundle_discount_pct,omitempty"`
	ActiveDiscounts           []ActiveDiscount `json:"active_discounts,omitempty"`
	UserActiveDiscounts       []ActiveDiscount `json:"user_active_discounts,omitempty"`
	IncludedGameCount         int32            `json:"included_game_count,omitempty"`
	HidePrice                 bool             `json:"hide_price,omitempty"`
	IncludedInSubscriptions   bool             `json:"included_in_subscriptions,omitempty"`
	FreeOrUnowned             bool             `json:"free_or_unowned,omitempty"`
	IsFreeToKeep              bool             `json:"is_free_to_keep,omitempty"`
	RequiresShipping          bool             `json:"requires_shipping,omitempty"`
	UserCanPurchase           bool             `json:"user_can_purchase,omitempty"`
	UserCanPurchaseAsGift     bool             `json:"user_can_purchase_as_gift,omitempty"`
	IsCommercialLicense       bool             `json:"is_commercial_license,omitempty"`
	ShouldSuppressDiscountPct bool             `json:"should_suppress_discount_pct,omitempty"`
}

// ActiveDiscount describes one currently-running discount on a purchase option.
type ActiveDiscount struct {
	DiscountAmount      int64  `json:"discount_amount,string,omitempty"`
	DiscountPct         int32  `json:"discount_pct,omitempty"`
	DiscountEndDate     int64  `json:"discount_end_date,omitempty"`
	DiscountDescription string `json:"discount_description,omitempty"`
}

// IncludedItems lists what a bundle or package contains.
type IncludedItems struct {
	IncludedApps     []StoreItem `json:"included_apps,omitempty"`
	IncludedPackages []StoreItem `json:"included_packages,omitempty"`
}

// RelatedItems references parents/children of this item (DLC -> base game, etc.).
type RelatedItems struct {
	ParentAppID uint32 `json:"parent_appid,omitempty"`
}
