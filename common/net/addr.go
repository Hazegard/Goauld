package net

import (
	"fmt"
	"net"
)

// validateIpAddress verify if the provided string is a valid IP address
// returns an error if not valid
func validateIpAddress(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	return nil
}

// validateCIDR verify if the provided string is a valid CIDR range
// returns an error if not valid
func validateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR: %s", cidr)
	}
	return nil
}

// IsValidPort verify if the given port is a valid port number
func IsValidPort(port int) bool {
	return port >= 0 && port <= 65535 || port == -1
}

// IsValidCIDR verify if the provided string is a valid CIDR range
func IsValidCIDR(cidr string) bool {
	return validateCIDR(cidr) == nil
}

// IsValidIP verify if the provided string is a valid IP address
func IsValidIP(ip string) bool {
	return validateIpAddress(ip) == nil
}

// IsValidIP verify if the provided string is a valid IP address or a valid CIDR range
func IsIPorCIDR(ip string) bool {
	return IsValidIP(ip) || IsValidCIDR(ip)
}
