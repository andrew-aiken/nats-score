package dns

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aaiken/nats-score/pkg/checks"
	"github.com/aaiken/nats-score/pkg/settings"

	"github.com/miekg/dns"
)

// The Definition configures the behavior of the DNS check
// it implements the "check" interface
type Definition struct {
	Server         string `json:"server"`                  // The IP of the DNS server to query
	Fqdn           string `json:"fqdn"`                    // The FQDN of the host you are looking up
	ExpectedResult string `json:"expected_result"`         // The expected IP of the host you are looking up
	Port           string `json:"port" default:"53"`       // The port of the DNS server
	RecordType     string `json:"record_type" default:"A"` // The type of DNS record to query
}

type Input struct{}

// Run a single instance of the check
func (d *Definition) Run(ctx context.Context, input map[string]any, static settings.StaticConf) checks.Results {
	// Initialize empty result
	result := checks.Results{Timestamp: time.Now()}

	recordType, ok := dns.StringToType[d.RecordType]
	if !ok {
		result.Message = fmt.Sprintf("Unknown record type: %s", d.RecordType)
		return result
	}

	// Setup for dns query
	var msg dns.Msg
	fqdn := dns.Fqdn(d.Fqdn)
	msg.SetQuestion(fqdn, recordType)

	// Make it obey timeout via deadline
	// TODO: change this to be relative to the parent context's timeout
	deadctx, cancel := context.WithDeadline(ctx, time.Now().Add(20*time.Second))
	defer cancel()

	// Send the query
	in, err := dns.ExchangeContext(deadctx, &msg, fmt.Sprintf("%s:%s", d.Server, d.Port))
	if err != nil {
		result.Message = fmt.Sprintf("Problem sending query to %s : %s", d.Server, err)
		return result
	}

	// Check if we got any records
	if len(in.Answer) < 1 {
		result.Message = fmt.Sprintf("No records received from %s", d.Server)
		return result
	}

	// Loop through results and check for correct match
	for _, answer := range in.Answer {
		if answer.Header().Rrtype != recordType {
			continue
		}

		// Extract record value by parsing the string format: name\tttl\tclass\ttype\tvalue
		parts := strings.SplitN(answer.String(), "\t", 5)

		if len(parts) >= 5 && strings.Trim(parts[4], "\"") == d.ExpectedResult {
			result.Passed = true
			return result
		}
	}

	// If we reach here no records matched expected IP and check fails
	result.Message = "Incorrect Records Returned"
	return result
}
