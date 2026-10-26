package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"golang.org/x/net/html"
)

// searchDDG uses the public non-JavaScript HTML endpoint. It never opens a
// browser, solves challenges, follows result links, paginates or sends a key.
func searchDDG(ctx context.Context, cfg *config.Config, query string, limit int) ([]searchResult, error) {
	endpoint, err := searchProviderEndpoint(cfg, "ddg")
	if err != nil {
		return nil, fmt.Errorf("invalid DuckDuckGo search URL")
	}
	q := endpoint.Query()
	q.Set("q", query)
	endpoint.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("invalid DuckDuckGo search request")
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "uiai-engine/agent-search (+https://github.com/WPUIAI/uiai-engine)")
	body, status, err := searchBody(req)
	if err != nil {
		return nil, err
	}
	results, challenge, err := parseDDGResults(body, normalizeSearchLimit(limit))
	if status == http.StatusAccepted || challenge {
		return nil, &upstreamSearchError{status: status, message: "DuckDuckGo requested verification; automated retries and challenge bypass are disabled", setupReason: "provider_challenge"}
	}
	if err != nil {
		return nil, err
	}
	return results, nil
}

func ddgAttr(node *html.Node, key string) string {
	for _, a := range node.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func ddgClass(node *html.Node, class string) bool {
	for _, s := range strings.Fields(ddgAttr(node, "class")) {
		if s == class {
			return true
		}
	}
	return false
}
func ddgText(node *html.Node) string {
	var out strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		if n.Type == html.TextNode {
			out.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(node)
	return strings.Join(strings.Fields(out.String()), " ")
}
func ddgFind(node *html.Node, class string) *html.Node {
	if ddgClass(node, class) {
		return node
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if found := ddgFind(c, class); found != nil {
			return found
		}
	}
	return nil
}
func ddgCitation(raw string) string {
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	ddgHost := func(host string) bool { return host == "duckduckgo.com" || host == "html.duckduckgo.com" }
	if ddgHost(strings.ToLower(u.Hostname())) && u.Path == "/l/" {
		// Query decoding occurs once; never fetch the tracking redirect.
		raw = u.Query().Get("uddg")
		u, err = url.Parse(raw)
		if err != nil {
			return ""
		}
	}
	if u.Scheme != "https" && u.Scheme != "http" || u.Hostname() == "" || u.User != nil {
		return ""
	}
	if ddgHost(strings.ToLower(u.Hostname())) && (u.Path == "/y.js" || u.Path == "/l/") {
		return ""
	}
	return u.String()
}
func parseDDGResults(body []byte, limit int) ([]searchResult, bool, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("DuckDuckGo returned invalid HTML")
	}
	results := []searchResult{}
	challenge, noResults := false, false
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, inAd bool) {
		if n.Type == html.ElementNode {
			if n.Data == "form" && (ddgAttr(n, "id") == "challenge-form" || strings.Contains(ddgAttr(n, "action"), "/anomaly.js")) || ddgClass(n, "anomaly-modal") {
				challenge = true
			}
			if ddgClass(n, "no-results") || ddgClass(n, "no-results__message") || ddgClass(n, "result--no-result") {
				noResults = true
			}
			inAd = inAd || ddgClass(n, "result--ad")
			if !inAd && ddgClass(n, "result") {
				link := ddgFind(n, "result__a")
				if link != nil && len(results) < limit {
					target := ddgCitation(ddgAttr(link, "href"))
					title := ddgText(link)
					if target != "" && title != "" {
						snippet := ""
						if node := ddgFind(n, "result__snippet"); node != nil {
							snippet = ddgText(node)
						}
						results = append(results, sanitizeSearchResult(searchResult{Title: title, URL: target, Description: snippet, Source: "DuckDuckGo"}))
					}
				}
				// Each result card is handled once; continue the document walk for
				// challenge markers even when the caller's result limit is reached.
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inAd)
		}
	}
	walk(doc, false)
	if challenge {
		return nil, true, nil
	}
	if len(results) == 0 && !noResults {
		return nil, false, fmt.Errorf("DuckDuckGo result markup was not recognized; no empty success inferred")
	}
	return results, false, nil
}
