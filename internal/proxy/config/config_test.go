package config

import (
	"net"
	"testing"
)

func TestParseIP(t *testing.T) {
	ip := "127.0.0.1"
	err := net.ParseIP(ip)
	t.Log(err)
}
