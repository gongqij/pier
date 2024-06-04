package config

const (
	tcpReverseProxys          = "tcp_reverse_proxys"
	tcpReverseProxysListenKey = "listen"
	tcpReverseProxysServerKey = "server"
	tcpReverseProxysAppIdKey  = "app_id"

	httpListen                  = "http.listen"
	httpHttp2Enable             = "http.http2_enable"
	httpRequestMaxContentLength = "http.request.max_content_length"
	httpRequestTimeout          = "http.request.timeout"
	httpRequestAllowOrigins     = "http.request.allow_origins"
	httpRemoteAddress           = "http.remote.address"
	httpRemoteReconnectTime     = "http.remote.reconnect_time"

	securityTLSEnable   = "security.tls.enable"
	securityTlsCAPath   = "security.tls.tlsCAPath"
	securityTlsCertPath = "security.tls.tlsCertPath"
	securityTlsPrivPath = "security.tls.tlsPrivPath"
	securityTlsDomain   = "security.tls.tlsDomain"
)
