package runner

import (
	"NetScan/internal/agent/domain"
	"fmt"
)

type Factory struct {
	httpRunner *HTTPRunner
	pingRunner *PingRunner
	dnsRunner  *DNSRunner
	tcpRunner  *TCPRunner
}

func NewFactory(http *HTTPRunner, ping *PingRunner, dns *DNSRunner, tcp *TCPRunner) *Factory {
	return &Factory{
		httpRunner: http,
		pingRunner: ping,
		dnsRunner:  dns,
		tcpRunner:  tcp,
	}
}

func (f *Factory) GetRunner(checkType domain.CheckType) (Runner, error) {
	switch checkType {
	case domain.HTTPCheck, domain.HTTPSCheck:
		return f.httpRunner, nil
	case domain.PingCheck:
		return f.pingRunner, nil
	case domain.DNSCheck:
		return f.dnsRunner, nil
	case domain.TCPCheck:
		return f.tcpRunner, nil
	default:
		return nil, fmt.Errorf("unknown check type: %s", checkType)
	}
}
