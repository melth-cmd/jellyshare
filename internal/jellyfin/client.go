package jellyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	userID     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type User struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

// FetchAndSetUserID fetches the first available user and stores the ID
func (c *Client) FetchAndSetUserID(ctx context.Context) error {
	resp, err := c.doRequest(ctx, http.MethodGet, "/Users", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch users: %d: %s", resp.StatusCode, string(body))
	}

	var users []User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return fmt.Errorf("failed to decode users: %w", err)
	}

	if len(users) == 0 {
		return fmt.Errorf("no users found in Jellyfin")
	}

	c.userID = users[0].ID
	return nil
}

type ItemInfo struct {
	ID                string        `json:"Id"`
	Name              string        `json:"Name"`
	Overview          string        `json:"Overview,omitempty"`
	Taglines          []string      `json:"Taglines,omitempty"`
	Type              string        `json:"Type"`
	RunTimeTicks      int64         `json:"RunTimeTicks,omitempty"`
	ImageTags         ImageTags     `json:"ImageTags,omitempty"`
	BackdropImageTags []string      `json:"BackdropImageTags,omitempty"`
	SeriesName        string        `json:"SeriesName,omitempty"`
	SeasonName        string        `json:"SeasonName,omitempty"`
	IndexNumber       int           `json:"IndexNumber,omitempty"`
	ParentIndexNumber int           `json:"ParentIndexNumber,omitempty"`
	ProductionYear    int           `json:"ProductionYear,omitempty"`
	PremiereDate      string        `json:"PremiereDate,omitempty"`
	OfficialRating    string        `json:"OfficialRating,omitempty"`
	CommunityRating   float64       `json:"CommunityRating,omitempty"`
	CriticRating      int           `json:"CriticRating,omitempty"`
	Genres            []string      `json:"Genres,omitempty"`
	Studios           []StudioInfo  `json:"Studios,omitempty"`
	People            []PersonInfo  `json:"People,omitempty"`
	MediaSources      []MediaSource `json:"MediaSources,omitempty"`
	Width             int           `json:"Width,omitempty"`
	Height            int           `json:"Height,omitempty"`
}

type ImageTags struct {
	Primary string `json:"Primary,omitempty"`
	Logo    string `json:"Logo,omitempty"`
	Thumb   string `json:"Thumb,omitempty"`
}

type StudioInfo struct {
	Name string `json:"Name"`
	ID   string `json:"Id"`
}

type PersonInfo struct {
	Name string `json:"Name"`
	ID   string `json:"Id"`
	Role string `json:"Role,omitempty"`
	Type string `json:"Type"`
}

type MediaSource struct {
	ID        string `json:"Id"`
	Name      string `json:"Name,omitempty"`
	Container string `json:"Container,omitempty"`
	// Path is the file on the Jellyfin host. Only its extension is ever used - it
	// names a download - and the path itself never leaves the backend.
	Path                 string        `json:"Path,omitempty"`
	Size                 int64         `json:"Size,omitempty"`
	Bitrate              int           `json:"Bitrate,omitempty"`
	SupportsDirectPlay   bool          `json:"SupportsDirectPlay"`
	SupportsDirectStream bool          `json:"SupportsDirectStream"`
	SupportsTranscoding  bool          `json:"SupportsTranscoding"`
	MediaStreams         []MediaStream `json:"MediaStreams,omitempty"`
}

type MediaStream struct {
	Type         string `json:"Type"`
	Index        int    `json:"Index"`
	Codec        string `json:"Codec,omitempty"`
	Language     string `json:"Language,omitempty"`
	Width        int    `json:"Width,omitempty"`
	Height       int    `json:"Height,omitempty"`
	BitRate      int    `json:"BitRate,omitempty"`
	Channels     int    `json:"Channels,omitempty"`
	SampleRate   int    `json:"SampleRate,omitempty"`
	DisplayTitle string `json:"DisplayTitle,omitempty"`
	IsDefault    bool   `json:"IsDefault,omitempty"`
	IsForced     bool   `json:"IsForced,omitempty"`
	// IsTextSubtitleStream separates srt/ass/mov_text, which can be converted to
	// WebVTT, from image formats like PGS that can only be rendered into the picture.
	IsText bool `json:"IsTextSubtitleStream,omitempty"`
}

type PlaybackInfo struct {
	MediaSources  []PlaybackMediaSource `json:"MediaSources"`
	PlaySessionId string                `json:"PlaySessionId"`
}

type PlaybackMediaSource struct {
	ID                   string `json:"Id"`
	Name                 string `json:"Name,omitempty"`
	Container            string `json:"Container,omitempty"`
	TranscodingUrl       string `json:"TranscodingUrl,omitempty"`
	DirectStreamUrl      string `json:"DirectStreamUrl,omitempty"`
	SupportsDirectPlay   bool   `json:"SupportsDirectPlay"`
	SupportsDirectStream bool   `json:"SupportsDirectStream"`
	SupportsTranscoding  bool   `json:"SupportsTranscoding"`
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.AuthHeader())
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) GetItem(ctx context.Context, itemID string) (*ItemInfo, error) {
	if c.userID == "" {
		return nil, fmt.Errorf("user ID not set - call FetchAndSetUserID first")
	}

	path := fmt.Sprintf("/Users/%s/Items/%s", c.userID, itemID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jellyfin API returned %d: %s", resp.StatusCode, string(body))
	}

	var item ItemInfo
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &item, nil
}

// SearchItem is one row in the admin library picker. It carries only what the
// picker shows; the share itself is built from the Jellyfin item id.
type SearchItem struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Year           int    `json:"year,omitempty"`
	SeriesName     string `json:"seriesName,omitempty"`
	SeasonName     string `json:"seasonName,omitempty"`
	SeasonNumber   int    `json:"seasonNumber,omitempty"`
	EpisodeNumber  int    `json:"episodeNumber,omitempty"`
	Overview       string `json:"overview,omitempty"`
	RuntimeSeconds int64  `json:"runtimeSeconds,omitempty"`
	HasPoster      bool   `json:"hasPoster"`
}

// SearchItems looks through the user's Jellyfin library. An empty query lists
// recently added items instead, so the picker opens with something to choose.
func (c *Client) SearchItems(ctx context.Context, query string, limit int) ([]SearchItem, error) {
	if c.userID == "" {
		return nil, fmt.Errorf("user ID not set - call FetchAndSetUserID first")
	}
	if limit <= 0 {
		limit = 20
	}

	params := url.Values{}
	params.Set("Recursive", "true")
	params.Set("IncludeItemTypes", "Movie,Series")
	params.Set("Fields", "Overview,ProductionYear,RunTimeTicks,SeriesName,SeasonName,IndexNumber,ParentIndexNumber,ImageTags")
	params.Set("ImageTypeLimit", "1")
	params.Set("EnableImageTypes", "Primary")
	params.Set("Limit", strconv.Itoa(limit))
	if query != "" {
		params.Set("SearchTerm", query)
		params.Set("SortBy", "SortName")
	} else {
		params.Set("SortBy", "DateCreated")
		params.Set("SortOrder", "Descending")
	}

	path := fmt.Sprintf("/Users/%s/Items?%s", c.userID, params.Encode())
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jellyfin API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []ItemInfo `json:"Items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return searchItemsFrom(result.Items), nil
}

func searchItemsFrom(items []ItemInfo) []SearchItem {
	rows := make([]SearchItem, 0, len(items))
	for _, item := range items {
		row := SearchItem{
			ID:             item.ID,
			Name:           item.Name,
			Type:           item.Type,
			Year:           item.ProductionYear,
			SeriesName:     item.SeriesName,
			SeasonName:     item.SeasonName,
			Overview:       item.Overview,
			RuntimeSeconds: TicksToSeconds(item.RunTimeTicks),
			HasPoster:      item.ImageTags.Primary != "",
		}
		switch item.Type {
		case "Episode":
			row.SeasonNumber = item.ParentIndexNumber
			row.EpisodeNumber = item.IndexNumber
		case "Season":
			row.SeasonNumber = item.IndexNumber
		}
		rows = append(rows, row)
	}
	return rows
}

// BrowseSeasons returns a series' seasons in order, Specials included.
func (c *Client) BrowseSeasons(ctx context.Context, seriesID string) ([]SearchItem, error) {
	return c.browseChildren(ctx, seriesID, "Season", "IndexNumber", "Overview,ImageTags,IndexNumber", false)
}

// BrowseEpisodes returns one season's episodes in order.
func (c *Client) BrowseEpisodes(ctx context.Context, seasonID string) ([]SearchItem, error) {
	return c.browseChildren(ctx, seasonID, "Episode", "IndexNumber", "Overview,RunTimeTicks,ImageTags,IndexNumber,ParentIndexNumber", false)
}

// BrowseSeriesEpisodes returns every episode under a series, for the fallback
// when Jellyfin has episodes but no Season objects.
func (c *Client) BrowseSeriesEpisodes(ctx context.Context, seriesID string) ([]SearchItem, error) {
	return c.browseChildren(ctx, seriesID, "Episode", "ParentIndexNumber,IndexNumber", "Overview,RunTimeTicks,ImageTags,IndexNumber,ParentIndexNumber", true)
}

func (c *Client) browseChildren(ctx context.Context, parentID, itemType, sortBy, fields string, recursive bool) ([]SearchItem, error) {
	if c.userID == "" {
		return nil, fmt.Errorf("user ID not set - call FetchAndSetUserID first")
	}

	params := url.Values{}
	params.Set("ParentId", parentID)
	if recursive {
		params.Set("Recursive", "true")
	}
	params.Set("IncludeItemTypes", itemType)
	params.Set("SortBy", sortBy)
	params.Set("SortOrder", "Ascending")
	params.Set("Fields", fields)
	params.Set("ImageTypeLimit", "1")
	params.Set("EnableImageTypes", "Primary")

	path := fmt.Sprintf("/Users/%s/Items?%s", c.userID, params.Encode())
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jellyfin API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []ItemInfo `json:"Items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode library items: %w", err)
	}
	return searchItemsFrom(result.Items), nil
}

func (c *Client) GetPlaybackInfo(ctx context.Context, itemID string) (*PlaybackInfo, error) {
	path := fmt.Sprintf("/Items/%s/PlaybackInfo", itemID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jellyfin API returned %d: %s", resp.StatusCode, string(body))
	}

	var info PlaybackInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &info, nil
}

// GetSubtitleURL returns the WebVTT rendering of a text subtitle stream.
func (c *Client) GetSubtitleURL(itemID, mediaSourceID string, index int) string {
	if mediaSourceID == "" {
		mediaSourceID = itemID
	}
	return fmt.Sprintf("%s/Videos/%s/%s/Subtitles/%d/Stream.vtt", c.baseURL, itemID, mediaSourceID, index)
}

// GetOriginalFileURL returns the untouched source file of a media source. Static
// means Jellyfin serves the file as it is on disk, with Range support, rather than
// starting a transcode.
func (c *Client) GetOriginalFileURL(itemID, mediaSourceID string) string {
	if mediaSourceID == "" {
		mediaSourceID = itemID
	}
	return fmt.Sprintf("%s/Videos/%s/stream?static=true&mediaSourceId=%s", c.baseURL, itemID, mediaSourceID)
}

func (c *Client) GetPosterURL(itemID string) string {
	return fmt.Sprintf("%s/Items/%s/Images/Primary", c.baseURL, itemID)
}

func (c *Client) GetBackdropURL(itemID string) string {
	return fmt.Sprintf("%s/Items/%s/Images/Backdrop", c.baseURL, itemID)
}

func (c *Client) GetLogoURL(itemID string) string {
	return fmt.Sprintf("%s/Items/%s/Images/Logo", c.baseURL, itemID)
}

func (c *Client) GetThumbURL(itemID string) string {
	return fmt.Sprintf("%s/Items/%s/Images/Thumb", c.baseURL, itemID)
}

func (c *Client) VerifyConnection(ctx context.Context) error {
	resp, err := c.doRequest(ctx, http.MethodGet, "/System/Info/Public", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Jellyfin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jellyfin returned status %d", resp.StatusCode)
	}

	return nil
}

// AuthHeader returns the standard MediaBrowser Authorization header value.
// Jellyfin 10.11 removed legacy auth (X-Emby-Token header, api_key query param).
func (c *Client) AuthHeader() string {
	return `MediaBrowser Client="jellyshare", Device="jellyshare-backend", DeviceId="jellyshare-backend", Version="1.0", Token="` + c.apiKey + `"`
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

// UserID is the Jellyfin user the backend acts as, resolved at startup.
func (c *Client) UserID() string {
	return c.userID
}

// TicksToSeconds converts Jellyfin runtime ticks to seconds
func TicksToSeconds(ticks int64) int64 {
	return ticks / 10000000
}

// EpisodeInfo contains basic info for an episode in a season
type EpisodeInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IndexNumber int    `json:"indexNumber"`
	// Overview is the episode's synopsis. Of the episode queries only
	// GetEpisodeListFor asks Jellyfin for it.
	Overview       string `json:"overview,omitempty"`
	RuntimeSeconds int64  `json:"runtimeSeconds,omitempty"`
	HasPoster      bool   `json:"hasPoster"`
	PremiereDate   string `json:"premiereDate,omitempty"`
	// SeasonNumber is the episode's season (Jellyfin's ParentIndexNumber), set
	// on every list; the page names it only on a Series link, which spans seasons.
	SeasonNumber int `json:"seasonNumber,omitempty"`
	// SizeBytes is the episode's file, which a download hands over as it is. Only
	// GetEpisodeListFor fills it.
	SizeBytes int64 `json:"sizeBytes,omitempty"`
}

// ContainsEpisode is the check that keeps an episode request inside its share.
func ContainsEpisode(episodes []EpisodeInfo, id string) bool {
	for _, ep := range episodes {
		if ep.ID == id {
			return true
		}
	}
	return false
}

// GetEpisodesFor returns the episodes a Season or Series item contains, in
// running order. Any other item type has none.
func (c *Client) GetEpisodesFor(ctx context.Context, itemType, itemID string) ([]EpisodeInfo, error) {
	return c.episodes(ctx, itemType, itemID, "")
}

// GetEpisodeListFor is GetEpisodesFor with each episode's file size and
// overview, for the list a viewer sees. Jellyfin leaves both out of a list
// unless asked, and the size costs every episode's media sources, so the
// checks that only need the ids - is this episode in the share - do without.
func (c *Client) GetEpisodeListFor(ctx context.Context, itemType, itemID string) ([]EpisodeInfo, error) {
	return c.episodes(ctx, itemType, itemID, "MediaSources,Overview")
}

func (c *Client) episodes(ctx context.Context, itemType, itemID, fields string) ([]EpisodeInfo, error) {
	items, err := c.episodeItems(ctx, itemType, itemID, fields)
	if err != nil {
		return nil, err
	}
	episodes := make([]EpisodeInfo, 0, len(items))
	for _, item := range items {
		var size int64
		if len(item.MediaSources) > 0 {
			size = item.MediaSources[0].Size
		}
		episodes = append(episodes, EpisodeInfo{
			SizeBytes:      size,
			ID:             item.ID,
			Name:           item.Name,
			IndexNumber:    item.IndexNumber,
			SeasonNumber:   item.ParentIndexNumber,
			Overview:       item.Overview,
			RuntimeSeconds: TicksToSeconds(item.RunTimeTicks),
			HasPoster:      item.ImageTags.Primary != "",
			PremiereDate:   item.PremiereDate,
		})
	}
	return episodes, nil
}

// GetEpisodeFilesFor returns the episode items with their media sources and
// paths, so the files can be fetched without a lookup per episode. It is the same query,
// so a ZIP holds exactly the episodes the page lists.
func (c *Client) GetEpisodeFilesFor(ctx context.Context, itemType, itemID string) ([]ItemInfo, error) {
	return c.episodeItems(ctx, itemType, itemID, "MediaSources,Path")
}

// episodeItems is the one episode query: every episode below a Season or
// Series, ordered by season then episode. A single recursive request rather
// than one per season - this runs on unauthenticated endpoints, and a 20-season
// show would otherwise cost 21 upstream calls.
func (c *Client) episodeItems(ctx context.Context, itemType, itemID, fields string) ([]ItemInfo, error) {
	if itemType != "Season" && itemType != "Series" {
		return nil, nil
	}
	if c.userID == "" {
		return nil, fmt.Errorf("user ID not set - call FetchAndSetUserID first")
	}

	path := fmt.Sprintf(
		"/Users/%s/Items?ParentId=%s&Recursive=true&IncludeItemTypes=Episode&SortBy=ParentIndexNumber,IndexNumber&SortOrder=Ascending",
		c.userID, itemID)
	if fields != "" {
		path += "&Fields=" + fields
	}
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jellyfin API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []ItemInfo `json:"Items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode episodes: %w", err)
	}
	return result.Items, nil
}

// GetSeasonEpisodes returns all episodes in a season
func (c *Client) GetSeasonEpisodes(ctx context.Context, seasonID string) ([]EpisodeInfo, error) {
	if c.userID == "" {
		return nil, fmt.Errorf("user ID not set - call FetchAndSetUserID first")
	}

	// Use the Items endpoint with ParentId filter
	path := fmt.Sprintf("/Users/%s/Items?ParentId=%s&SortBy=IndexNumber&SortOrder=Ascending", c.userID, seasonID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jellyfin API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []ItemInfo `json:"Items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	episodes := make([]EpisodeInfo, 0, len(result.Items))
	for _, item := range result.Items {
		if item.Type != "Episode" {
			continue
		}
		ep := EpisodeInfo{
			ID:           item.ID,
			Name:         item.Name,
			IndexNumber:  item.IndexNumber,
			SeasonNumber: item.ParentIndexNumber,
			Overview:     item.Overview,
			PremiereDate: item.PremiereDate,
			HasPoster:    item.ImageTags.Primary != "",
		}
		if item.RunTimeTicks > 0 {
			ep.RuntimeSeconds = TicksToSeconds(item.RunTimeTicks)
		}
		episodes = append(episodes, ep)
	}

	return episodes, nil
}

// GetSeriesSeasons returns all seasons in a series
func (c *Client) GetSeriesSeasons(ctx context.Context, seriesID string) ([]EpisodeInfo, error) {
	if c.userID == "" {
		return nil, fmt.Errorf("user ID not set - call FetchAndSetUserID first")
	}

	path := fmt.Sprintf("/Users/%s/Items?ParentId=%s&SortBy=IndexNumber&SortOrder=Ascending", c.userID, seriesID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jellyfin API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []ItemInfo `json:"Items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	seasons := make([]EpisodeInfo, 0, len(result.Items))
	for _, item := range result.Items {
		if item.Type != "Season" {
			continue
		}
		s := EpisodeInfo{
			ID:          item.ID,
			Name:        item.Name,
			IndexNumber: item.IndexNumber,
			Overview:    item.Overview,
			HasPoster:   item.ImageTags.Primary != "",
		}
		seasons = append(seasons, s)
	}

	return seasons, nil
}
