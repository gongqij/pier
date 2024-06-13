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
	HTTPCancelTimeout    time.Duration
	HTTPRetryDuration    time.Duration
	HTTPAllFailedLimit   int
	HTTPMaxContentLength int64
	HTTPRequestTimeout   time.Duration
	HTTPAllowOrigins     []string

	RemoteReconnectTime time.Duration
	RemoteAddress       []string
	RemoteHttpPort      []int
	RemoteTcpPort       []int

	SecurityTLSEnable   bool
	SecurityTLSCAPath   string
	SecurityTLSCertPath string
	SecurityTLSPrivPath string
	SecurityTLSDomain   string
}

type ReverseProxyConfig struct {
	Listen int64
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
			if key == tcpReverseProxysAppIdKey {
				appId, ok = value.(string)
				if !ok {
					return nil, fmt.Errorf("invalid type for key tcp_reverse_proxys.app_id")
				}
			}
		}
		reverseProxys[listen] = ReverseProxyConfig{
			Listen: listen,
			AppId:  appId,
		}
	}

	httpListenPort := vip.GetInt64(httpListen)
	if cerr := checkPortInt(httpListenPort); cerr != nil {
		return nil, fmt.Errorf("invalid port range for key http.listen, err: %s", cerr.Error())
	}
	http2Enable := vip.GetBool(httpHttp2Enable)
	httpCancelTimeoutStr := vip.GetString(httpCancelTimeout)
	hct, derr := time.ParseDuration(httpCancelTimeoutStr)
	if derr != nil {
		return nil, fmt.Errorf("parse http cancel timeout error: %s", derr.Error())
	}
	httpRetryDurationStr := vip.GetString(httpRetryDuration)
	hrd, derr := time.ParseDuration(httpRetryDurationStr)
	if derr != nil {
		return nil, fmt.Errorf("parse http retry duration error: %s", derr.Error())
	}
	hafl := vip.GetInt(httpAllFailedLimit)
	httpAllowOriginsStrSlice := vip.GetStringSlice(httpRequestAllowOrigins)
	httpRequestTimeoutStr := vip.GetString(httpRequestTimeout)
	duration, derr := time.ParseDuration(httpRequestTimeoutStr)
	if derr != nil {
		return nil, fmt.Errorf("parse request timeout error: %s", derr.Error())
	}
	httpMaxContentLength := vip.GetInt64(httpRequestMaxContentLength)
	remoteReconnectTimeStr := vip.GetString(remoteReconnectTime)
	remoteAddressStrSlice := vip.GetStringSlice(remoteAddress)
	remoteHttpPortIntSlice := vip.GetIntSlice(remoteHttpPort)
	remoteTcpPortIntSlice := vip.GetIntSlice(remoteTcpPort)
	if len(remoteAddressStrSlice) == 0 || len(remoteHttpPortIntSlice) == 0 || len(remoteTcpPortIntSlice) == 0 {
		return nil, fmt.Errorf("http.remote.address, http.remote.http_port, http.remote.tcp_port can't be empty")
	}
	if (len(remoteAddressStrSlice) != len(remoteHttpPortIntSlice)) || (len(remoteAddressStrSlice) != len(remoteTcpPortIntSlice)) {
		return nil, fmt.Errorf("the length of array must be same for http.remote.address, http.remote.http_port, http.remote.tcp_port")
	}

	reconectTime, derr := time.ParseDuration(remoteReconnectTimeStr)
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
		HTTPCancelTimeout:    hct,
		HTTPRetryDuration:    hrd,
		HTTPAllFailedLimit:   hafl,
		HTTPMaxContentLength: httpMaxContentLength,
		HTTPRequestTimeout:   duration,
		HTTPAllowOrigins:     httpAllowOriginsStrSlice,
		RemoteAddress:        remoteAddressStrSlice,
		RemoteTcpPort:        remoteTcpPortIntSlice,
		RemoteHttpPort:       remoteHttpPortIntSlice,
		RemoteReconnectTime:  reconectTime,
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
