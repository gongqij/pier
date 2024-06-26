package http

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"time"

	json "github.com/json-iterator/go"
)

type node struct {
	ip       string
	httpPort int

	httpUrl string // join ip and httpPort
	alive   bool

	confIndex int // 地址在配置文件里的索引位置
}

// newNode creates an instance of node.
func newNode(ip string, httpPort int, isHttps bool, confIndex int) (*node, error) {
	var scheme string
	if isHttps {
		scheme = "https://"
	} else {
		scheme = "http://"
	}

	httpAddress, jerr := joinIPAndPort(ip, httpPort)
	if jerr != nil {
		return nil, fmt.Errorf("invalid address format, err: %v", jerr.Error())
	}

	return &node{
		ip:        ip,
		httpPort:  httpPort,
		httpUrl:   fmt.Sprintf("%s%s%s", scheme, httpAddress, uri),
		alive:     true,
		confIndex: confIndex,
	}, nil
}

func joinIPAndPort(ip string, port int) (string, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	// 检查 IP 地址是 IPv4 还是 IPv6
	if parsedIP.To4() != nil {
		// IPv4
		return fmt.Sprintf("%s:%d", ip, port), nil
	} else {
		// IPv6
		return fmt.Sprintf("[%s]:%d", ip, port), nil
	}
}

func (h *Http) selectNodeUrl() (index int, url string, err error) {
	var tempNodes []*node
	for _, v := range h.nodes {
		tempNodes = append(tempNodes, v)
	}
	sort.SliceStable(tempNodes, func(i, j int) bool {
		if tempNodes[i].alive && !tempNodes[j].alive {
			return true
		} else if !tempNodes[i].alive && tempNodes[j].alive {
			return false
		}
		return false
	})

	if tempNodes[0].alive {
		return tempNodes[0].confIndex, tempNodes[0].httpUrl, nil
	}

	return 0, "", errors.New("all nodes are bad, please check it")
}

func (h *Http) reconnectNode(confIndex int) {
	defer h.wg.Done()

	timer := time.NewTimer(h.conf.RemoteReconnectTime)
	defer timer.Stop()

	url := h.nodes[confIndex].httpUrl
	data := newJsonData(1, httpReconnectMsg, httpReconnectMsg, []byte("ping"))
	body, err := json.Marshal(data)
	if err != nil {
		h.log.Error(err)
	}

	h.log.Info("start to reconnect node " + url)

	for {
		select {
		case <-h.stopped:
			return
		case <-timer.C:
			req, nerr := newHttpRequest(http.MethodPost, url, body)
			if nerr != nil {
				h.log.Error(nerr)
			}

			response, err := h.httpCli.Do(req)
			if err != nil {
				h.log.Debug(err)
				timer.Reset(h.conf.RemoteReconnectTime)
				break
			}

			if response != nil && response.StatusCode == http.StatusOK {
				b, _ := io.ReadAll(response.Body)
				h.log.Debug("reconnection node body: ", string(b))
				response.Body.Close()
				h.nodes[confIndex].alive = true
				h.log.Info("node " + url + " Reconnect Success!")
				return
			}
			response.Body.Close()
			h.log.Debug("node " + url + " Reconnect failed, will try later")

			timer.Reset(h.conf.RemoteReconnectTime)
		}
	}
}
