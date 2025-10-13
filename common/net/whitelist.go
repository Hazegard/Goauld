package net

// IsIPAllowed check if the IP is in the allowed list.
func IsIPAllowed(ip string, allowedIPs []string) bool {
	if len(allowedIPs) == 0 {
		return true
	}
	for _, allowedIP := range allowedIPs {
		if ip == allowedIP {
			return true
		}
	}

	return false
}
