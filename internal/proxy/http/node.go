package http

import (
	"errors"
	"io"
	"net/http"
	"sort"
	"time"

	json "github.com/json-iterator/go"
)

type node struct {
	url       string
	alive     bool
	confIndex int // 在配置文件里的索引位置
}

// newNode creates an instance of node, the format of url is ip:port.
func newNode(url string, isHttps bool, confIndex int) *node {
	var scheme string

	if isHttps {
		scheme = "https://"
	} else {
		scheme = "http://"
	}

	return &node{
		url:       scheme + url,
		alive:     true,
		confIndex: confIndex,
	}
}

// the returned result:
func (h *Http) selectNodeUrl() (index int, url string, err error) {
	var tempNodes []*node
	for _, v := range h.nodes {
		tempNodes = append(tempNodes, v)
	}
	sort.SliceStable(tempNodes, func(i, j int) bool {
		if tempNodes[i].alive && !tempNodes[j].alive {
			return tempNodes[i].alive
		} else if !tempNodes[i].alive && tempNodes[j].alive {
			return tempNodes[j].alive
		}
		return false
	})

	if tempNodes[0].alive {
		return tempNodes[0].confIndex, tempNodes[0].url, nil
	}

	return 0, "", errors.New("all nodes are bad, please check it")
}

func (h *Http) reconnectNode(confIndex int) {
	//todo 可能存在goroutine泄露，需要改一下，每个url只有一个reconnectNode gorroutine
	defer h.wg.Done()

	timer := time.NewTimer(h.conf.HTTPReconnectTime)
	defer timer.Stop()

	url := h.nodes[confIndex].url
	data := newJsonData(1, httpReconnectMsg, httpReconnectMsg, []byte("ping"))
	body, err := json.Marshal(data)
	if err != nil {
		h.log.Error(err)
	}

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
				//todo 日志级别可能有点高
				h.log.Error(err)
				timer.Reset(h.conf.HTTPReconnectTime)
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

			timer.Reset(h.conf.HTTPReconnectTime)
		}
	}
}
