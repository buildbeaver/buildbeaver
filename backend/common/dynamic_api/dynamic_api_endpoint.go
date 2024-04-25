package dynamic_api

import (
	"fmt"
	"net"
	"net/url"
	"runtime"
	"strings"
	"sync"
)

// Endpoint is a configuration value specifying an endpoint to connect to the Dynamic API, for
// use by dynamic build jobs. It must contain a valid URL.
type Endpoint string

func (s Endpoint) String() string {
	return string(s)
}

const dockerBridgeInterfaceName = "docker0"

// GetDockerDynamicEndpoint returns the endpoint URL that docker-based dynamic jobs should use to contact
// the dynamic API, based on the endpoint that non-docker based jobs should use.
// This will only differ in the case where the endpoint is on the local host machine; in this case
// a special address must be used for docker, dependent on the host operating system.
func GetDockerDynamicEndpoint(nonDockerEndpoint Endpoint) (Endpoint, error) {
	nonDockerURL, err := url.Parse(nonDockerEndpoint.String())
	if err != nil {
		return "", err
	}
	if !IsLocalhost(nonDockerURL.Hostname()) {
		// If not local server then docker jobs should use the same endpoint as non-docker jobs
		return nonDockerEndpoint, nil
	}

	// Non-docker endpoint is on the local host. The address to use inside a docker container to get
	// to the host depends on which OS is being used.
	var dockerEndpoint string
	switch runtime.GOOS {
	case "windows", "darwin":
		// Windows and Mac run docker inside a VM and have a special name for getting to the real host
		dockerEndpoint = "http://host.docker.internal"
	case "linux":
		// Use the docker bridge host IP address to get back to the dynamic API server running on the host
		dockerBridgeIP, err := GetDockerBridgeInterfaceIPv4Address()
		if err != nil {
			return "", err
		}
		dockerEndpoint = "http://" + dockerBridgeIP
	default:
		// Assume other operating systems are the same as Linux
		dockerBridgeIP, err := GetDockerBridgeInterfaceIPv4Address()
		if err != nil {
			return "", err
		}
		dockerEndpoint = "http://" + dockerBridgeIP
	}

	if nonDockerURL.Port() != "" {
		dockerEndpoint += ":" + nonDockerURL.Port()
	}

	return Endpoint(dockerEndpoint), nil
}

// IsLocalhost returns true if the specified host name or IP address refers to the local network interface,
// i.e. "localhost", "127.0.0.1" or equivalent IPv6 address.
func IsLocalhost(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1", "0:0:0:0:0:0:0:1", "0000:0000:0000:0000:0000:0000:0000:0001":
		return true
	default:
		return false
	}
}

var (
	cachedBridgeAddress      string
	cachedBridgeAddressMutex sync.Mutex
)

// GetDockerBridgeInterfaceIPv4Address returns an IPv4 address (normally "172.17.0.1") for the default docker
// bridge network interface. Returns an error if no such IPv4 address can be found.
// The IP address is cached when found so this function can be called frequently with no performance cost.
func GetDockerBridgeInterfaceIPv4Address() (string, error) {
	cachedBridgeAddressMutex.Lock()
	defer cachedBridgeAddressMutex.Unlock()

	if cachedBridgeAddress == "" {
		addr, err := GetNetworkInterfaceIPv4Address(dockerBridgeInterfaceName)
		if err != nil {
			return "", err
		}
		cachedBridgeAddress = addr
	}

	return cachedBridgeAddress, nil
}

// GetNetworkInterfaceIPv4Address returns an IPv4 address (e.g. "192.168.0.1") for the network interface
// with the specified name. Returns an error if no such IPv4 address can be found.
func GetNetworkInterfaceIPv4Address(interfaceName string) (string, error) {
	inter, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return "", fmt.Errorf("error: interface '%s' not found", interfaceName)
	}
	addresses, err := inter.Addrs()
	if err != nil {
		return "", fmt.Errorf("error listing network addresses for interface %s: %w", inter.Name, err)
	}
	// Look for an IPv4 address
	for _, addr := range addresses {
		// Strip any netmask suffix
		splitStr := strings.SplitN(addr.String(), "/", 2)
		if len(splitStr) >= 1 {
			ip := splitStr[0]
			err = checkIPV4(ip)
			if err != nil {
				continue // this is not an IPv4 address so move on to the next address
			}
			// found a valid IPv4 address
			return ip, nil
		}
	}
	return "", fmt.Errorf("error: no IPv4 address found for interface '%s'", interfaceName)
}

// checkIPV4 checks that the supplied string contains a valid IPv4 address (e.g. "192.168.0.1").
// Returns an error if the string is not a valid IP
// This code is adapted from the Go version 1.20 net/netip package parseIPv4() function
func checkIPV4(s string) error {
	var fields [4]uint8
	var val, pos int
	var digLen int // number of digits in current octet
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			if digLen == 1 && val == 0 {
				return fmt.Errorf("IPv4 field has octet with leading zero")
			}
			val = val*10 + int(s[i]) - '0'
			digLen++
			if val > 255 {
				return fmt.Errorf("IPv4 field has value >255")
			}
		} else if s[i] == '.' {
			// .1.2.3
			// 1.2.3.
			// 1..2.3
			if i == 0 || i == len(s)-1 || s[i-1] == '.' {
				return fmt.Errorf("IPv4 field must have at least one digit")
			}
			// 1.2.3.4.5
			if pos == 3 {
				return fmt.Errorf("IPv4 address too long")
			}
			fields[pos] = uint8(val)
			pos++
			val = 0
			digLen = 0
		} else {
			return fmt.Errorf("unexpected character")
		}
	}
	if pos < 3 {
		return fmt.Errorf("IPv4 address too short")
	}
	fields[3] = uint8(val)
	return nil
}
