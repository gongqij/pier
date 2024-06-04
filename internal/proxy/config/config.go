package config

import (
	"fmt"
	"github.com/spf13/viper"
	"net"
	"strconv"
	"strings"
	"time"
)

type ProxyConfig struct {
	ReverseProxys map[int64]ReverseProxyConfig // key is port

	HTTPPort             int64
	HTTP2Enable          bool
	HTTPMaxContentLength int64
	HTTPRequestTimeout   time.Duration
	HTTPAllowOrigins     []string
	HTTPRemoteAddress    []string
	HTTPReconnectTime    time.Duration

	SecurityTLSEnable   bool
	SecurityTLSCAPath   string
	SecurityTLSCertPath string
	SecurityTLSPrivPath string
	SecurityTLSDomain   string
}

type ReverseProxyConfig struct {
	Listen int64
	Server string
	AppId  string
}

func LoadProxyConfig(cfgFilePath string) (*ProxyConfig, error) {
	vip := viper.New()
	vip.SetConfigFile(cfgFilePath)
	vip.SetConfigType("toml")

	if err := vip.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s failed, err: %s", cfgFilePath, err.Error())
	}

	reverseProxys := make(map[int64]ReverseProxyConfig)
	trpInter := vip.Get(tcpReverseProxys)
	trpSlice, tok := trpInter.([]interface{})
	if !tok {
		return nil, fmt.Errorf("invalid type for key tcp_reverse_proxys")
	}
	for _, item := range trpSlice {
		proxyConfig, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid type for key tcp_reverse_proxys")
		}
		var (
			listen int64
			server string
			appId  string
		)
		for key, value := range proxyConfig {
			if key == tcpReverseProxysListenKey {
				listen, ok = value.(int64)
				if !ok {
					return nil, fmt.Errorf("invalid type for key tcp_reverse_proxys.listen")
				}
				if cerr := checkPortInt(listen); cerr != nil {
					return nil, fmt.Errorf("invalid port range for key tcp_reverse_proxys.server, err: %s", cerr.Error())
				}
			}
			if key == tcpReverseProxysServerKey {
				server, ok = value.(string)
				if !ok {
					return nil, fmt.Errorf("invalid type for key tcp_reverse_proxys.server")
				}
				if _, _, cerr := checkAddress(server); cerr != nil {
					return nil, fmt.Errorf("invalid address format for key tcp_reverse_proxys.server, err: %s", cerr.Error())
				}
			}
			if key == tcpReverseProxysAppIdKey {
				appId, ok = value.(string)
				if !ok {
					return nil, fmt.Errorf("invalid type for key tcp_reverse_proxys.app_id")
				}
			}
		}
		reverseProxys[listen] = ReverseProxyConfig{
			Listen: listen,
			Server: server,
			AppId:  appId,
		}
	}

	httpListenPort := vip.GetInt64(httpListen)
	if cerr := checkPortInt(httpListenPort); cerr != nil {
		return nil, fmt.Errorf("invalid port range for key http.listen, err: %s", cerr.Error())
	}
	http2Enable := vip.GetBool(httpHttp2Enable)
	httpAllowOriginsStrSlice := vip.GetStringSlice(httpRequestAllowOrigins)
	httpRequestTimeoutStr := vip.GetString(httpRequestTimeout)
	duration, derr := time.ParseDuration(httpRequestTimeoutStr)
	if derr != nil {
		return nil, fmt.Errorf("parse request timeout error: %s", derr.Error())
	}
	httpMaxContentLength := vip.GetInt64(httpRequestMaxContentLength)
	httpRemoteAddressStrSlice := vip.GetStringSlice(httpRemoteAddress)
	if len(httpRemoteAddressStrSlice) == 0 {
		return nil, fmt.Errorf("http.remote.address can't be empty")
	}
	for _, addr := range httpRemoteAddressStrSlice {
		if _, _, cerr := checkAddress(addr); cerr != nil {
			return nil, fmt.Errorf("invalid address format for key http.remote.address")
		}
	}
	httpReconnectTimeStr := vip.GetString(httpRemoteReconnectTime)
	reconectTime, derr := time.ParseDuration(httpReconnectTimeStr)
	if derr != nil {
		return nil, fmt.Errorf("parse reconnect time error: %s", derr.Error())
	}

	securityTlsEnable := vip.GetBool(securityTLSEnable)
	securityTlsCAPathStr := vip.GetString(securityTlsCAPath)
	securityTlsCertPathStr := vip.GetString(securityTlsCertPath)
	securityTlsPrivPathStr := vip.GetString(securityTlsPrivPath)
	securityTlsDomainStr := vip.GetString(securityTlsDomain)

	return &ProxyConfig{
		ReverseProxys:        reverseProxys,
		HTTPPort:             httpListenPort,
		HTTP2Enable:          http2Enable,
		HTTPMaxContentLength: httpMaxContentLength,
		HTTPRequestTimeout:   duration,
		HTTPAllowOrigins:     httpAllowOriginsStrSlice,
		HTTPRemoteAddress:    httpRemoteAddressStrSlice,
		HTTPReconnectTime:    reconectTime,
		SecurityTLSEnable:    securityTlsEnable,
		SecurityTLSCAPath:    securityTlsCAPathStr,
		SecurityTLSCertPath:  securityTlsCertPathStr,
		SecurityTLSPrivPath:  securityTlsPrivPathStr,
		SecurityTLSDomain:    securityTlsDomainStr,
	}, nil
}

func checkAddress(addr string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid address type")
	}
	if strings.TrimSpace(host) == "" {
		return "", 0, fmt.Errorf("empty host")
	}
	nodePort, _ := strconv.Atoi(portStr)
	if nodePort > 65535 || nodePort <= 1024 {
		return "", 0, fmt.Errorf("invalid port range")
	}
	return host, nodePort, nil
}

func checkPortInt(port int64) error {
	if port > 65535 || port <= 1024 {
		return fmt.Errorf("invalid port range")
	}
	return nil
}
