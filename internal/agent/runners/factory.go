package runner

import (
	"NetScan/internal/agent/domain"
	"fmt"
)

type Factory struct {
	httpRunner *HTTPRunner
	pingRunner *PingRunner
	dnsRunner  *DNSRunner
}

func NewFactory(http *HTTPRunner, ping *PingRunner, dns *DNSRunner) *Factory {
	return &Factory{
		httpRunner: http,
		pingRunner: ping,
		dnsRunner:  dns,
	}
}

func (f *Factory) GetRunner(checkType domain.CheckType) (Runner, error) {
	switch checkType {
	case domain.HTTPCheck:
		return f.httpRunner, nil
	case domain.PingCheck:
		return f.pingRunner, nil
	case domain.DNSCheck:
		return f.dnsRunner, nil
	default:
		return nil, fmt.Errorf("unknown check type: %s", checkType)
	}
}
