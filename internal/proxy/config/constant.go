package config

const (
	tcpReverseProxys          = "tcp_reverse_proxys"
	tcpReverseProxysListenKey = "listen"
	tcpReverseProxysAppIdKey  = "app_id"

	httpListen                  = "http.listen"
	httpHttp2Enable             = "http.http2_enable"
	httpCancelTimeout           = "http.cancel_timeout"
	httpRetryDuration           = "http.retry_duration"
	httpAllFailedLimit          = "http.all_failed_limit"
	httpRequestMaxContentLength = "http.request.max_content_length"
	httpRequestTimeout          = "http.request.timeout"
	httpRequestAllowOrigins     = "http.request.allow_origins"

	remoteReconnectTime = "remote.reconnect_time"
	remoteAddress       = "remote.address"
	remoteHttpPort      = "remote.http_port"
	remoteTcpPort       = "remote.tcp_port"

	securityTLSEnable   = "security.tls.enable"
	securityTlsCAPath   = "security.tls.tlsCAPath"
	securityTlsCertPath = "security.tls.tlsCertPath"
	securityTlsPrivPath = "security.tls.tlsPrivPath"
	securityTlsDomain   = "security.tls.tlsDomain"
)
