package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	json "github.com/json-iterator/go"
	"github.com/meshplus/pier/internal/proxy/common"
	"github.com/meshplus/pier/internal/proxy/config"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func newClient(conf *config.ProxyConfig) *http.Client {
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		DisableKeepAlives:     false,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		TLSHandshakeTimeout:   10 * time.Second, //TLS安全连接握手超时时间
		ExpectContinueTimeout: 1 * time.Second,  //发送完请求到接收到响应头的超时时间
	}
	if conf.SecurityTLSEnable {
		pool := x509.NewCertPool()

		tlsCAPath := conf.SecurityTLSCAPath
		tlsCertPath := conf.SecurityTLSCertPath
		tlsCertPrivPath := conf.SecurityTLSPrivPath

		caCrt, err := os.ReadFile(tlsCAPath)
		if err != nil {
			panic(fmt.Sprintf("read tlsCA from %s failed, err: %s", tlsCAPath, err.Error()))
		}
		pool.AppendCertsFromPEM(caCrt)

		cliCrt, err := tls.LoadX509KeyPair(tlsCertPath, tlsCertPrivPath)
		if err != nil {
			panic(fmt.Sprintf("read tlspeerCert from %s and %s failed, err: %s", tlsCertPath, tlsCertPrivPath, err.Error()))
		}

		cfg := &tls.Config{
			ServerName:   conf.SecurityTLSDomain,
			RootCAs:      pool,
			Certificates: []tls.Certificate{cliCrt},
		}

		if !conf.SecurityTLSBidirectionalCertAuthEnable {
			cfg.InsecureSkipVerify = true
		}

		tr.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			tlsConn, err := tls.DialWithDialer(&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}, network, addr, cfg)
			if err != nil {
				return nil, err
			}
			return tlsConn, nil
		}
	}
	return &http.Client{Transport: tr}
}

func newHttpRequest(method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// sendHttpRequest sends http request to remote proxy.
func (h *Http) sendHttpRequest(data *common.Data) {

	// 1. 选择一个可用节点进行发送
	confIndex, httpUrl, serr := h.selectNodeUrl()
	if serr != nil {
		h.log.Errorf("drop message because can't select a http node, err: %v", serr)
		return
	}

	if data.Typ == common.TcpProxyTypeConnect {
		h.log.Infof("prepare to connect target pier, its http url is: %s", h.nodes[confIndex].httpUrl)
	}
	jsonReq := newJsonData(h.requestID, data.Uuid, data.Typ, data.Content)
	jsonReqRaw, _ := json.Marshal(jsonReq)

	h.log.Debugf("receive data from tcp, send http request { requestId: %v, uuid: %v, httpUrl: %v}", jsonReq.Id, jsonReq.TcpUUid, httpUrl)

	req, err := newHttpRequest(http.MethodPost, httpUrl, jsonReqRaw)
	if err != nil {
		h.log.Errorf("drop message because failed to create POST request: %v", err)
		return
	}

	// 尝试获取发送锁
	hasGotLock := h.redisCli.SendLock()
	if !hasGotLock {
		// 未拿到发送锁，说明自己不是主pier，丢弃消息并释放锁
		h.log.Warning("drop message because the local node is not master")
		h.redisCli.SendUnlock()
		return
	}
	// 拿到发送锁，说明自己是主pier，可以发送http请求
	resp, rerr := h.httpCli.Do(req)
	if rerr != nil || resp == nil || resp.StatusCode != http.StatusOK {
		// 2. 第一次 http 请求发送失败，重发
		if rerr != nil {
			h.log.Errorf("retry send http request to %s, err: %v", httpUrl, rerr)
		} else {
			h.log.Errorf("retry send http request to %s, err: status code is %v", httpUrl, resp.StatusCode)
		}
		time.Sleep(100 * time.Millisecond)
		if resp, rerr = h.httpCli.Do(req); rerr != nil {
			// 3. 第二次 http 请求发送失败，尝试切换 remote httpUrl 发送
			if strings.Contains(rerr.Error(), "connection refused") {
				h.nodes[confIndex].alive = false
				h.wg.Add(1)
				go h.reconnectNode(confIndex)
			}

			resp, rerr = h.resendHttpRequest(http.MethodPost, jsonReqRaw)
			if rerr != nil || resp == nil {
				h.log.Errorf("drop message because failed to send http request, err: %v", rerr)
				h.redisCli.SendUnlock()
				return
			}

			if resp.StatusCode != http.StatusOK {
				h.log.Errorf("drop message because failed to send http request, status code is: %v", resp.StatusCode)
				h.redisCli.SendUnlock()
				return
			}
		}
	}
	defer resp.Body.Close()

	// 更新 requestId
	h.requestID++
	if h.requestID > maxRequestId {
		h.requestID = 1
	}

	// 释放发送锁
	h.redisCli.SendUnlock()

	// 打印 http response debug 日志
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.log.Errorf("failed to read response body: %v", err)
		return
	}
	var respJson JsonData
	if uerr := json.Unmarshal(body, &respJson); uerr != nil {
		h.log.Errorf("failed to unmarshal response body: %v", uerr)
		return
	}

	h.log.Debugf("receive http response { requestId: %v, uuid: %v, httpUrl: %v, statusCode: %v, Err: %v}", respJson.Id, respJson.TcpUUid, httpUrl, resp.StatusCode, respJson.Err)
}

func (h *Http) resendHttpRequest(method string, jsonReqRaw []byte) (*http.Response, error) {
	for {
		select {
		case <-h.stopped:
			return nil, errors.New("http has stopped, not resend")
		default:
			confIndex, url, serr := h.selectNodeUrl()
			if serr != nil {
				return nil, serr
			}

			req, err := newHttpRequest(method, url, jsonReqRaw)
			if err != nil {
				return nil, fmt.Errorf("failed to create POST request: %v", err)
			}

			resp, rerr := h.httpCli.Do(req)
			if rerr == nil && resp.StatusCode == http.StatusOK {
				h.log.Infof("choose httpUrl %v to resend", url)
				return resp, rerr
			}

			if rerr != nil && strings.Contains(rerr.Error(), "connection refused") {
				h.nodes[confIndex].alive = false
				h.wg.Add(1)
				go h.reconnectNode(confIndex)
				time.Sleep(200 * time.Millisecond)
			}
		}
	}
}
