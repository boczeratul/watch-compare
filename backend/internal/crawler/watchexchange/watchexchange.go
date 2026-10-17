// Package watchexchange indexes for-sale posts on Reddit's r/Watchexchange
// (https://www.reddit.com/r/Watchexchange), a peer-to-peer marketplace of private sellers.
//
// Data comes from Reddit's listing JSON, newest first: /r/Watchexchange/new (100 posts per page,
// cursor "after"; Reddit serves at most ~1000 posts of a listing, i.e. 10 pages, which covers a
// few days of the sub). With REDDIT_CLIENT_ID / REDDIT_CLIENT_SECRET set, requests go to
// oauth.reddit.com with an app-only token (client-credentials grant), which Reddit's Data API
// terms expect from servers and which is not blocked from cloud IP ranges; without them the
// public www.reddit.com/…/new.json endpoint is used.
//
// Post shape (sub rules): title "[WTS] Tudor Black Bay 58 79030N - $3,100 shipped", link flair
// "WTS" / "WTS/WTT" / "WTT" / "WTB" / "Sold". Only WTS posts are indexed; WTB, trade-only,
// sold, removed and deleted posts are skipped. The asking price is read from the title, then from
// the body. Photos come from Reddit galleries or link previews; imgur album links are kept in the
// attributes since the API does not expand them.
package watchexchange

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	subreddit = "Watchexchange"
	publicURL = "https://www.reddit.com"
	oauthURL  = "https://oauth.reddit.com"
	tokenURL  = "https://www.reddit.com/api/v1/access_token"
	pageSize  = 100
	maxPages  = 10 // Reddit stops paginating a listing after ~1000 items
	// Reddit asks API clients for a unique, descriptive User-Agent; generic browser UAs get throttled.
	userAgent = "server:watch-compare:v1.0 (+https://github.com/boczeratul/watch-compare)"
)

// Source implements crawler.Source.
type Source struct {
	mu     sync.Mutex
	token  string
	expiry time.Time
}

// Key returns the source key.
func (*Source) Key() string { return "watchexchange" }

// Marketplace marks r/Watchexchange as multi-country: sellers post from anywhere.
func (*Source) Marketplace() bool { return true }

// Crawl walks /new until the listing ends, the page budget runs out or a page brings nothing new.
func (s *Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	pages := min(env.MaxPages, maxPages)
	seen := map[string]bool{}
	after := ""
	for page := 1; page <= pages; page++ {
		body, err := s.get(ctx, env, after)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		posts, next, err := ParseListing(body)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		if len(posts) == 0 && page == 1 {
			return fmt.Errorf("no posts on first page (API changed?)")
		}
		fresh := 0
		for _, p := range posts {
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			fresh++
			l, ok := ToListing(p)
			if !ok {
				continue
			}
			if err := emit(l); err != nil {
				return err
			}
		}
		if next == "" || fresh == 0 {
			return nil
		}
		after = next
	}
	return nil
}

func (s *Source) get(ctx context.Context, env *crawler.Env, after string) ([]byte, error) {
	q := url.Values{"limit": {strconv.Itoa(pageSize)}, "raw_json": {"1"}}
	if after != "" {
		q.Set("after", after)
	}
	base, path := publicURL, "/r/"+subreddit+"/new.json"
	token := ""
	if env.Cfg.RedditClientID != "" && env.Cfg.RedditClientSecret != "" {
		t, err := s.accessToken(ctx, env)
		if err != nil {
			return nil, err
		}
		base, path, token = oauthURL, "/r/"+subreddit+"/new", t
	}
	req, err := http.NewRequest(http.MethodGet, base+path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := env.Fetcher.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: http %d", req.URL.Path, resp.StatusCode)
	}
	return body, nil
}

func (s *Source) accessToken(ctx context.Context, env *crawler.Env) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && time.Now().Before(s.expiry.Add(-2*time.Minute)) {
		return s.token, nil
	}
	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(env.Cfg.RedditClientID+":"+env.Cfg.RedditClientSecret)))
	resp, err := env.Fetcher.Client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reddit token: http %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("reddit token: empty access_token")
	}
	s.token = tok.AccessToken
	s.expiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return s.token, nil
}

// Post is the subset of a Reddit link ("t3") the adapter reads.
type Post struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Title             string  `json:"title"`
	Selftext          string  `json:"selftext"`
	Author            string  `json:"author"`
	Permalink         string  `json:"permalink"`
	URL               string  `json:"url"`
	LinkFlairText     string  `json:"link_flair_text"`
	CreatedUTC        float64 `json:"created_utc"`
	Score             int     `json:"score"`
	NumComments       int     `json:"num_comments"`
	Stickied          bool    `json:"stickied"`
	Over18            bool    `json:"over_18"`
	RemovedByCategory *string `json:"removed_by_category"`
	IsGallery         bool    `json:"is_gallery"`
	GalleryData       *struct {
		Items []struct {
			MediaID string `json:"media_id"`
		} `json:"items"`
	} `json:"gallery_data"`
	MediaMetadata map[string]struct {
		Status string `json:"status"`
		E      string `json:"e"`
		S      struct {
			U string `json:"u"`
		} `json:"s"`
	} `json:"media_metadata"`
	Preview *struct {
		Images []struct {
			Source struct {
				URL string `json:"url"`
			} `json:"source"`
		} `json:"images"`
	} `json:"preview"`
}

// ParseListing decodes one page of a Reddit listing. Exported for fixture tests.
func ParseListing(body []byte) (posts []Post, after string, err error) {
	var resp struct {
		Kind string `json:"kind"`
		Data struct {
			After    string `json:"after"`
			Children []struct {
				Kind string `json:"kind"`
				Data Post   `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("decode listing: %w", err)
	}
	if resp.Kind != "Listing" {
		return nil, "", fmt.Errorf("unexpected kind %q", resp.Kind)
	}
	for _, c := range resp.Data.Children {
		if c.Kind == "t3" {
			posts = append(posts, c.Data)
		}
	}
	return posts, resp.Data.After, nil
}

var (
	tagRe = regexp.MustCompile(`\[[^\]]{1,20}\]`) // "[WTS]", "[WTS/WTT]", "[Price drop]"
	// A price with its currency: "$3,100", "US$ 950", "C$1200", "€2.450", "£900", "3100 USD", "$4.5k".
	priceRe = regexp.MustCompile(`(?i)(?:(US\s?\$|USD\s?\$?|C\$|CA\$|CAD\s?\$?|A\$|AU\$|AUD\s?\$?|HK\$|S\$|SGD\s?\$?|\$|€|£|¥)\s?(\d{1,3}(?:[,.]\d{3})+|\d+(?:\.\d+)?)(\s?k\b)?)|(?:\b(\d{1,3}(?:[,.]\d{3})+|\d+(?:\.\d+)?)(\s?k)?\s?(USD|CAD|AUD|EUR|GBP|HKD|SGD|JPY|euros?)\b)`)
	// Body lines that state the asking price, preferred over the first amount (often "retail $X").
	askRe      = regexp.MustCompile(`(?i)\b(price|asking|ask|selling for|sale price|net to me|shipped)\b`)
	unwornRe   = regexp.MustCompile(`(?i)\b(unworn|brand new|bnib|new in box|nos|never worn)\b`)
	conusRe    = regexp.MustCompile(`(?i)\bconus\b|\bships? from (the )?us(a)?\b|\blocated in (the )?us(a)?\b`)
	imageExtRe = regexp.MustCompile(`(?i)\.(jpe?g|png|webp)(\?|$)`)
)

// ToListing maps a post onto a listing; ok is false for posts that are not an active sale.
func ToListing(p Post) (model.Listing, bool) {
	if p.ID == "" || p.Stickied || p.Over18 || (p.RemovedByCategory != nil && *p.RemovedByCategory != "") ||
		p.Author == "[deleted]" || p.Selftext == "[removed]" || p.Selftext == "[deleted]" {
		return model.Listing{}, false
	}
	if !forSale(p) {
		return model.Listing{}, false
	}
	title := normalize.CleanText(strings.Trim(tagRe.ReplaceAllString(p.Title, " "), " -–|:"))
	if title == "" {
		return model.Listing{}, false
	}
	l := model.Listing{
		ExternalID: p.ID,
		URL:        publicURL + p.Permalink,
		Title:      title,
		SellerName: "u/" + p.Author,
		SellerType: "private",
		Condition:  model.ConditionGood, // private resale: pre-owned unless the post says otherwise
	}
	if price, cur, ok := Price(p.Title); ok {
		l.Price, l.Currency = &price, cur
	} else if price, cur, ok := BodyPrice(p.Selftext); ok {
		l.Price, l.Currency = &price, cur
	}
	// The amount in the title would otherwise be read as a reference ("Tudor BB58 $3200" → 3200).
	l.ReferenceNumber = normalize.ExtractReference(priceRe.ReplaceAllString(title, " "))
	if unwornRe.MatchString(p.Title) {
		l.Condition = model.ConditionUnworn
	}
	if conusRe.MatchString(p.Title + "\n" + p.Selftext) {
		l.LocationCountry = "US"
	}
	body := normalize.CleanText(p.Selftext)
	if len([]rune(body)) > 2000 {
		body = string([]rune(body)[:2000]) + "…"
	}
	l.Description = body
	l.HasBox, l.HasPapers = normalize.DetectBoxPapers(p.Title + " " + p.Selftext)
	l.ImageURLs = images(p)

	attrs := map[string]any{"flair": p.LinkFlairText, "author": p.Author, "score": p.Score, "comments": p.NumComments}
	if p.CreatedUTC > 0 {
		attrs["postedAt"] = time.Unix(int64(p.CreatedUTC), 0).UTC().Format(time.RFC3339)
	}
	if album := albumLink(p); album != "" {
		attrs["album"] = album
	}
	l.Attributes, _ = json.Marshal(attrs)
	return l, true
}

// forSale accepts WTS posts (flair or title tag) and rejects WTB, trade-only and sold posts.
func forSale(p Post) bool {
	flair := strings.ToUpper(p.LinkFlairText)
	t := strings.ToUpper(p.Title)
	if strings.Contains(flair, "SOLD") || strings.Contains(t, "[SOLD]") || strings.Contains(t, "(SOLD)") {
		return false
	}
	switch {
	case strings.Contains(flair, "WTS"):
		return true
	case flair != "":
		return false // WTB, WTT, META, …
	}
	return strings.Contains(t, "WTS")
}

// Price returns the first currency-marked amount in s. Bare numbers are ignored: titles are
// full of references and years.
func Price(s string) (float64, string, bool) {
	for _, m := range priceRe.FindAllStringSubmatch(s, -1) {
		sym, num, k := m[1], m[2], m[3]
		if num == "" {
			num, k, sym = m[4], m[5], m[6]
		}
		v, ok := normalize.ParsePrice(num)
		if !ok {
			continue
		}
		if strings.TrimSpace(k) != "" {
			v *= 1000
		}
		if v < 20 || v > 5_000_000 {
			continue
		}
		return v, currency(sym), true
	}
	return 0, "", false
}

// BodyPrice prefers an amount on a line that names the asking price, then the first amount.
func BodyPrice(body string) (float64, string, bool) {
	lines := strings.Split(body, "\n")
	for _, ln := range lines {
		if askRe.MatchString(ln) {
			if v, c, ok := Price(ln); ok {
				return v, c, true
			}
		}
	}
	return Price(body)
}

func currency(sym string) string {
	s := strings.ToUpper(strings.ReplaceAll(sym, " ", ""))
	switch {
	case strings.HasPrefix(s, "C") || strings.HasPrefix(s, "CA"):
		return "CAD"
	case strings.HasPrefix(s, "A"):
		return "AUD"
	case strings.HasPrefix(s, "HK"):
		return "HKD"
	case strings.HasPrefix(s, "S"):
		return "SGD"
	case s == "€" || strings.HasPrefix(s, "EUR"):
		return "EUR"
	case s == "£" || s == "GBP":
		return "GBP"
	case s == "¥" || s == "JPY":
		return "JPY"
	}
	return "USD"
}

func images(p Post) []string {
	var out []string
	if p.GalleryData != nil {
		for _, it := range p.GalleryData.Items {
			if m, ok := p.MediaMetadata[it.MediaID]; ok && m.Status == "valid" && m.E == "Image" && m.S.U != "" {
				out = append(out, html.UnescapeString(m.S.U))
			}
		}
	}
	if len(out) == 0 && p.Preview != nil {
		for _, im := range p.Preview.Images {
			if u := html.UnescapeString(im.Source.URL); u != "" {
				out = append(out, u)
			}
		}
	}
	if len(out) == 0 && imageExtRe.MatchString(p.URL) {
		out = append(out, p.URL)
	}
	return out
}

func albumLink(p Post) string {
	u := strings.ToLower(p.URL)
	if strings.Contains(u, "imgur.com/a/") || strings.Contains(u, "imgur.com/gallery/") {
		return p.URL
	}
	return ""
}
