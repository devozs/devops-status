package providerverify

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

const esVerifyTimeout = 15 * time.Second

// VerifyElasticsearch pings the cluster URL using the official client.
func VerifyElasticsearch(ctx context.Context, clusterURL, username, password string) (latencyMs int, err error) {
	u := strings.TrimSpace(clusterURL)
	if u == "" {
		return 0, fmt.Errorf("url is required")
	}
	if parsed, err := url.Parse(u); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return 0, fmt.Errorf("url must be a valid http(s) URL")
	}

	cfg := elasticsearch.Config{
		Addresses: []string{u},
		Username:  strings.TrimSpace(username),
		Password:  password,
	}
	// Respect context via request-level options below
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return 0, fmt.Errorf("elasticsearch client: %w", err)
	}

	start := time.Now()
	res, err := es.Ping(es.Ping.WithContext(ctx))
	latencyMs = int(time.Since(start).Milliseconds())
	if err != nil {
		return latencyMs, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return latencyMs, fmt.Errorf("elasticsearch ping: %s", res.String())
	}
	return latencyMs, nil
}
