package icmp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aaiken/nats-score/pkg/checks"

	probing "github.com/prometheus-community/pro-bing"
)

type Definition struct {
	AllowPacketLoss bool   `default:"true"`        // Pass check based on received pings matching Count; if false, will use percent packet loss
	Count           int    `default:"1"`           // The number of ICMP requests to send per check
	Host            string `optiontype:"required"` // IP or hostname of the host to run the ICMP check against
	Percent         int    `default:"100"`         // Percent of packets needed to come back to pass the check
}

type Input struct{}

func (d *Definition) Run(ctx context.Context, input map[string]interface{}) checks.Results {
	// Initialize empty result
	result := checks.Results{Timestamp: time.Now()}

	// Create pinger
	pinger, err := probing.NewPinger(d.Host)
	if err != nil {
		result.Message = fmt.Sprintf("Error creating pinger: %s", err)
		return result
	}

	// Send ping
	pinger.Count = d.Count
	// TODO: change this to be relative to the parent context's timeout
	pinger.Timeout = 10 * time.Second
	_ = pinger.Run()

	stats := pinger.Statistics()

	details := make(map[string]string)
	// Check packet loss instead of count
	if !d.AllowPacketLoss {
		if stats.PacketLoss >= float64(d.Percent) {
			result.Message = "Not all pings made it back!"
			details["packetloss_percent"] = strconv.FormatFloat(stats.PacketLoss, 'f', -1, 64)
			result.Details = details
			return result
		}

		// If we make it here the check passes by percentage
		result.Passed = true
		return result
	}

	// Check for failure of ICMP
	if stats.PacketsRecv != d.Count {
		result.Message = "Not all pings made it back!"
		details["packets_received"] = fmt.Sprintf("%d", stats.PacketsRecv)
		details["packets_expected"] = fmt.Sprintf("%d", d.Count)
		result.Details = details
		return result
	}

	// If we make it here the check passes
	result.Passed = true

	return result
}
