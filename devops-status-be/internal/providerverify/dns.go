package providerverify

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const dnsVerifyTimeout = 10 * time.Second

// VerifyDNS resolves hostname using the system resolver or an optional nameserver (host or host:port).
func VerifyDNS(ctx context.Context, hostname, recordType, nameserver string) (latencyMs int, err error) {
	host := strings.TrimSpace(strings.TrimSuffix(hostname, "."))
	if host == "" {
		return 0, fmt.Errorf("hostname is required")
	}
	rt := strings.ToLower(strings.TrimSpace(recordType))
	if rt == "" {
		rt = "a"
	}
	ns := strings.TrimSpace(nameserver)

	start := time.Now()
	if ns != "" {
		err = verifyDNSCustom(ctx, host, rt, normalizeNameserver(ns))
	} else {
		err = verifyDNSSystem(ctx, host, rt)
	}
	latencyMs = int(time.Since(start).Milliseconds())
	return latencyMs, err
}

func normalizeNameserver(ns string) string {
	if strings.Contains(ns, ":") {
		return ns
	}
	return net.JoinHostPort(ns, "53")
}

func verifyDNSSystem(ctx context.Context, host, rt string) error {
	r := net.DefaultResolver
	switch rt {
	case "a":
		addrs, err := r.LookupHost(ctx, host)
		if err != nil {
			return err
		}
		if len(addrs) == 0 {
			return fmt.Errorf("no A records for %q", host)
		}
		return nil
	case "aaaa":
		ips, err := r.LookupIPAddr(ctx, host)
		if err != nil {
			return err
		}
		var v6 bool
		for _, ip := range ips {
			if ip.IP.To4() == nil && ip.IP.To16() != nil {
				v6 = true
				break
			}
		}
		if !v6 {
			return fmt.Errorf("no AAAA records for %q", host)
		}
		return nil
	case "cname":
		cname, err := r.LookupCNAME(ctx, host)
		if err != nil {
			return err
		}
		if cname == "" || strings.EqualFold(strings.TrimSuffix(cname, "."), strings.TrimSuffix(host, ".")) {
			return fmt.Errorf("no CNAME for %q", host)
		}
		return nil
	case "txt":
		txts, err := r.LookupTXT(ctx, host)
		if err != nil {
			return err
		}
		if len(txts) == 0 {
			return fmt.Errorf("no TXT records for %q", host)
		}
		return nil
	default:
		return fmt.Errorf("unsupported record_type %q (use a, aaaa, cname, txt)", rt)
	}
}

func verifyDNSCustom(ctx context.Context, host, rt, server string) error {
	ctx, cancel := context.WithTimeout(ctx, dnsVerifyTimeout)
	defer cancel()

	var qType uint16
	switch rt {
	case "a":
		qType = dns.TypeA
	case "aaaa":
		qType = dns.TypeAAAA
	case "cname":
		qType = dns.TypeCNAME
	case "txt":
		qType = dns.TypeTXT
	default:
		return fmt.Errorf("unsupported record_type %q (use a, aaaa, cname, txt)", rt)
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), qType)
	m.RecursionDesired = true

	c := &dns.Client{Net: "udp", Timeout: dnsVerifyTimeout}
	r, _, err := c.ExchangeContext(ctx, m, server)
	if err != nil {
		return err
	}
	if r == nil || r.Rcode != dns.RcodeSuccess {
		code := "unknown"
		if r != nil {
			if s, ok := dns.RcodeToString[r.Rcode]; ok {
				code = s
			}
		}
		return fmt.Errorf("dns query failed: %s", code)
	}
	if len(r.Answer) == 0 {
		return fmt.Errorf("no %s records for %q", strings.ToUpper(rt), host)
	}
	return nil
}
