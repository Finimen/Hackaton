package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/miekg/dns"
)

type DNSRunner struct {
	timeout time.Duration
}

func NewDNSRunner() *DNSRunner {
	return &DNSRunner{
		timeout: time.Second * 10,
	}
}

func (r *DNSRunner) Execute(ctx context.Context, target string, options map[string]interface{}) (map[string]interface{}, error) {
	recordType := getStringOption(options, "record_type", "A")
	server := getStringOption(options, "server", "8.8.8.8:53")
	timeout := getDurationOption(options, "timeout", r.timeout)

	client := &dns.Client{
		Timeout: timeout,
	}

	msg := dns.Msg{}
	msg.SetQuestion(dns.Fqdn(target), recordTypeToDNSType(recordType))

	response, rtt, err := client.ExchangeContext(ctx, &msg, server)
	if err != nil {
		return nil, fmt.Errorf("DNS query failed: %w", err)
	}

	if response.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("DNS error: %s", dns.RcodeToString[response.Rcode])
	}

	records := make([]string, 0)
	for _, answer := range response.Answer {
		records = append(records, answer.String())
	}

	result := map[string]interface{}{
		"records":          records,
		"server":           server,
		"response_time":    rtt.Milliseconds(),
		"answer_count":     len(response.Answer),
		"authority_count":  len(response.Ns),
		"additional_count": len(response.Extra),
		"record_type":      recordType,
	}

	if len(response.Answer) > 0 {
		if ttl := extractMinTTL(response.Answer); ttl > 0 {
			result["ttl"] = ttl
		}
	}

	return result, nil
}

func recordTypeToDNSType(recordType string) uint16 {
	switch recordType {
	case "A":
		return dns.TypeA
	case "AAAA":
		return dns.TypeAAAA
	case "MX":
		return dns.TypeMX
	case "NS":
		return dns.TypeNS
	case "TXT":
		return dns.TypeTXT
	case "CNAME":
		return dns.TypeCNAME
	case "SOA":
		return dns.TypeSOA
	case "PTR":
		return dns.TypePTR
	case "SRV":
		return dns.TypeSRV
	default:
		return dns.TypeA
	}
}

func extractMinTTL(answers []dns.RR) uint32 {
	if len(answers) == 0 {
		return 0
	}

	minTTL := answers[0].Header().Ttl
	for _, answer := range answers[1:] {
		if answer.Header().Ttl < minTTL {
			minTTL = answer.Header().Ttl
		}
	}
	return minTTL
}
