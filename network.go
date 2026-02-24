package main

import (
	"net"
	"strings"
)

func getLocalNetworks() []*net.IPNet {
	var networks []*net.IPNet
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil {
					networks = append(networks, ipnet)
				}
			}
		}
	}
	return networks
}

func isIPInNetworks(ipStr string, networks []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func extractHost(address string) string {
	if idx := strings.Index(address, "://"); idx >= 0 {
		address = address[idx+3:]
	}
	if idx := strings.Index(address, "/"); idx >= 0 {
		address = address[:idx]
	}
	return address
}

// getLocalIPs returns all non-loopback IPv4 addresses of this device.
// Works on Android and iOS without needing MAC addresses.
func getLocalIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					ips = append(ips, ip4.String())
				}
			}
		}
	}
	return ips
}

// FindThisDevice matches the current device in the router's client list by IP address.
// This works on both Android and iOS (no MAC address needed).
func FindThisDevice(clients []Client) *Client {
	myIPs := getLocalIPs()
	ipSet := make(map[string]bool, len(myIPs))
	for _, ip := range myIPs {
		ipSet[ip] = true
	}
	for i := range clients {
		if ipSet[clients[i].IP] {
			return &clients[i]
		}
	}
	return nil
}
