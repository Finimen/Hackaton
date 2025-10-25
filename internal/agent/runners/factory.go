package runner

import (
	"NetScan/internal/agent/domain"
	"fmt"
	"log"
)

type Factory struct {
	httpRunner *HTTPRunner
	pingRunner *PingRunner
	dnsRunner  *DNSRunner
	tcpRunner  *TCPRunner
}

func NewFactory(http *HTTPRunner, ping *PingRunner, dns *DNSRunner, tcp *TCPRunner) *Factory {
	log.Printf("🔧 DEBUG: Creating Factory - http: %p, ping: %p, dns: %p, tcp: %p",
		http, ping, dns, tcp)

	return &Factory{
		httpRunner: http,
		pingRunner: ping,
		dnsRunner:  dns,
		tcpRunner:  tcp,
	}
}

func (f *Factory) GetRunner(checkType domain.CheckType) (Runner, error) {
	log.Printf("🔧 DEBUG: GetRunner called with type: %s", checkType)
	log.Printf("🔧 DEBUG: Factory state - http: %p, ping: %p, dns: %p, tcp: %p",
		f.httpRunner, f.pingRunner, f.dnsRunner, f.tcpRunner)

	switch checkType {
	case domain.HTTPCheck, domain.HTTPSCheck:
		log.Printf("🔧 DEBUG: Returning HTTP runner: %p", f.httpRunner)
		if f.httpRunner == nil {
			return nil, fmt.Errorf("HTTP runner not initialized")
		}
		return f.httpRunner, nil
	case domain.PingCheck:
		log.Printf("🔧 DEBUG: Returning Ping runner: %p", f.pingRunner)
		if f.pingRunner == nil {
			return nil, fmt.Errorf("Ping runner not initialized")
		}
		return f.pingRunner, nil
	case domain.DNSCheck:
		log.Printf("🔧 DEBUG: Returning DNS runner: %p", f.dnsRunner)
		if f.dnsRunner == nil {
			return nil, fmt.Errorf("DNS runner not initialized")
		}
		return f.dnsRunner, nil
	case domain.TCPCheck:
		log.Printf("🔧 DEBUG: Returning TCP runner: %p", f.tcpRunner)
		if f.tcpRunner == nil {
			return nil, fmt.Errorf("TCP runner not initialized")
		}
		return f.tcpRunner, nil
	default:
		return nil, fmt.Errorf("unknown check type: %s", checkType)
	}
}
