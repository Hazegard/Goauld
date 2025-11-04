// Package net includes common network functions
package net

import (
	"fmt"
	"net"
	"strings"
)

// validateIPAddress verify if the provided string is a valid IP address
// returns an error if not valid.
func validateIPAddress(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	return nil
}

// validateCIDR verify if the provided string is a valid CIDR range
// returns an error if not valid.
func validateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR: %s", cidr)
	}

	return nil
}

// IsValidPort verify if the given port is a valid port number.
func IsValidPort(port int) bool {
	return port >= 0 && port <= 65535 || port == -1
}

// IsValidCIDR verify if the provided string is a valid CIDR range.
func IsValidCIDR(cidr string) bool {
	return validateCIDR(cidr) == nil
}

// IsValidIP verify if the provided string is a valid IP address.
func IsValidIP(ip string) bool {
	return validateIPAddress(ip) == nil
}

// IsIPorCIDR verify if the provided string is a valid IP address or a valid CIDR range.
func IsIPorCIDR(ip string) bool {
	return IsValidIP(ip) || IsValidCIDR(ip)
}

// IsLoopback returns whether the provided IP is a loopback address.
func IsLoopback(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false // invalid IP
	}

	return ip.IsLoopback()
}

// ExtractIP returns the IP address from an IP:port scheme.
func ExtractIP(addr string) (string, error) {
	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return strings.Split(ip, ":")[0], err
	}

	return ip, nil
}

func ParseCIDRs(input string) ([]net.IPNet, error) {
	var result []net.IPNet

	// Split by commas and trim spaces
	cidrList := strings.Split(input, ",")
	for _, cidr := range cidrList {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}

		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}

		result = append(result, *ipnet)
	}

	return result, nil
}
