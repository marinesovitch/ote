// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package system

import (
	"net"
)

// resolve host IP
func ResolveHostIP() (net.IP, error) {
	const RemoteAddress = "8.8.8.8:80"
	conn, err := net.Dial("udp", RemoteAddress)
	if err != nil {
		return net.IP{}, err
	}
	defer conn.Close()

	hostAddress := conn.LocalAddr().(*net.UDPAddr)
	return hostAddress.IP, nil
}

func IsLoopback(host string) (bool, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return false, err
	}
	for _, addr := range addrs {
		ipAddress := net.ParseIP(addr)
		if ipAddress.IsLoopback() {
			return true, nil
		}
	}
	return false, nil
}
