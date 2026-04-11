package providerverify

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bndr/gojenkins"
)

const jenkinsVerifyTimeout = 15 * time.Second

// VerifyJenkins uses the Jenkins REST client. authMethod is "none" or "basic" (password is often an API token).
func VerifyJenkins(ctx context.Context, rootURL, authMethod, username, password string) (latencyMs int, err error) {
	u := strings.TrimRight(strings.TrimSpace(rootURL), "/")
	if u == "" {
		return 0, fmt.Errorf("url is required")
	}
	if parsed, err := url.Parse(u); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return 0, fmt.Errorf("url must be a valid http(s) URL")
	}
	am := strings.ToLower(strings.TrimSpace(authMethod))
	if am == "" {
		am = "none"
	}
	if am != "none" && am != "basic" {
		return 0, fmt.Errorf("auth_method must be none or basic for jenkins")
	}
	user := strings.TrimSpace(username)
	if am == "basic" && (user == "" || password == "") {
		return 0, fmt.Errorf("basic auth requires username and password or API token")
	}

	j := gojenkins.CreateJenkins(nil, u, user, password)
	ctx, cancel := context.WithTimeout(ctx, jenkinsVerifyTimeout)
	defer cancel()

	start := time.Now()
	_, err = j.Init(ctx)
	latencyMs = int(time.Since(start).Milliseconds())
	if err != nil {
		return latencyMs, fmt.Errorf("jenkins: %w", err)
	}
	return latencyMs, nil
}
