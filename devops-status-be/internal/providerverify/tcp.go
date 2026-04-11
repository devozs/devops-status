package providerverify

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"
)

// VerifyTCP opens a TCP connection to host:port.
func VerifyTCP(ctx context.Context, host string, port int) (latencyMs int, err error) {
	if host == "" || port < 1 || port > 65535 {
		return 0, fmt.Errorf("host and valid port (1–65535) are required")
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	d := net.Dialer{Timeout: 5 * time.Second}
	c, err := d.DialContext(ctx, "tcp", addr)
	latencyMs = int(time.Since(start).Milliseconds())
	if err != nil {
		return latencyMs, err
	}
	_ = c.Close()
	return latencyMs, nil
}
