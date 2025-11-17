package blind

import (
	"Goauld/common/log"
	commonnet "Goauld/common/net"
	"fmt"
	"strconv"
	"strings"

	miekgDns "github.com/miekg/dns"
)

// TestDNSServer return whether the server DNS is reachable.
func TestDNSServer(serveur string, tld string) bool {
	p := 53
	var ip string
	split := strings.Split(serveur, ":")
	if len(split) == 2 {
		ip = split[0]
		var err error
		p, err = strconv.Atoi(split[1])
		if err != nil {
			log.Debug().Err(err).Str("Mode", "DNSSH").Str("Domain", serveur).Str("Port", split[1]).Msg("error parsing port, using 53 as default...")
			p = 53
		}
	} else {
		ip = serveur
	}
	log.Debug().Str("IP", ip).Str("Mode", "DNSSH").Int("Port", p).Msgf("Testing DNS server availability")

	// Define the domain and DNS server
	isOpen := commonnet.CheckHostPortAvailability("udp", ip, p)
	srv := fmt.Sprintf("%s:%d", ip, p)
	if !isOpen {
		log.Debug().Str("Mode", "DNSSH-ALT").Str("Server", srv).Msg("No DNS server found")

		return false
	}
	log.Debug().Str("Mode", "DNSSH-ALT").Str("Server", srv).Msg("Testing DNS server (TXT)")
	domain := "0000.ffff.0000." + tld
	// Prepare the DNS client
	client := new(miekgDns.Client)
	message := new(miekgDns.Msg)
	message.SetQuestion(miekgDns.Fqdn(domain), miekgDns.TypeTXT)
	// 3) Disable DNSSEC: DO bit = false
	//    (EDNS UDP buffer size 4096, DNSSEC OK = false)
	message.SetEdns0(4096, false)

	// Send the DNS query to the specified server
	response, _, err := client.Exchange(message, srv)
	if err != nil {
		log.Debug().Err(err).Str("Mode", "DNSSH-ALT").Str("Domain", domain).Str("Server", srv).Msg("error testing DNS server")

		return false
	}

	// Check if we received any TXT records in the response
	if len(response.Answer) > 0 {
		for _, ans := range response.Answer {
			if txtRecord, ok := ans.(*miekgDns.TXT); ok {
				log.Debug().Str("Record", txtRecord.String()).Str("Domain", domain).Str("Server", srv).Msg("record found")

				return true
			}
		}
	}
	log.Debug().Str("Domain", domain).Str("Server", srv).Msg("no record found")

	return false
}
