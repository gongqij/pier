package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/meshplus/pier/internal/proxy/common"
	"github.com/meshplus/pier/internal/proxy/config"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	//code "github.com/ultramesh/flato-common/errorcode"
)

type serverHandler struct {
	sendCh chan *common.Data

	conf *config.ProxyConfig
	log  logrus.FieldLogger
}

func newHttpServerHandler(sendCh chan *common.Data, log logrus.FieldLogger, conf *config.ProxyConfig) (http.Handler, error) {
	h := &serverHandler{
		sendCh: sendCh,
		conf:   conf,
		log:    log,
	}

	return newCorsHandler(h, conf.HTTPAllowOrigins), nil
}

func newCorsHandler(innerHandler http.Handler, allowedOrigins []string) http.Handler {
	// disable CORS support if user has not specified a custom CORS configuration
	if len(allowedOrigins) == 0 {
		return innerHandler
	}

	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"POST", "GET"},
		MaxAge:         600,
		AllowedHeaders: []string{"*"},
	})
	return c.Handler(innerHandler)
}

// send data to tcp layer
func (h *serverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	jsonMsg, err := toJsonData(r, h.conf.HTTPMaxContentLength)
	if err != nil {
		h.log.Errorf("drop message because decode json data failed, Err: %v", err)
		if werr := writeErrorResponse(w, jsonMsg.Id, jsonMsg.TcpUUid, jsonMsg.Typ, fmt.Errorf("to json data failed %v", err)); werr != nil {
			h.log.Errorf("failed to write error response, Err: %v", werr)
		}
		return
	}
	defer r.Body.Close()

	if jsonMsg.Typ == httpReconnectMsg {
		if werr := writeSuccessResponse(w, jsonMsg.Id, jsonMsg.TcpUUid, jsonMsg.Typ); werr != nil {
			h.log.Errorf("failed to write success response, Err: %v", werr)
		}
		return
	}

	h.sendCh <- &common.Data{
		Typ:     jsonMsg.Typ,
		Content: jsonMsg.Content,
		Uuid:    jsonMsg.TcpUUid,
	}

	if werr := writeSuccessResponse(w, jsonMsg.Id, jsonMsg.TcpUUid, jsonMsg.Typ); werr != nil {
		h.log.Errorf("failed to write success response, Err: %v", werr)
	}
}

func writeSuccessResponse(w http.ResponseWriter, requestId uint16, tcpuuid string, typ string) error {
	respVal, _ := json.Marshal(newJsonData(requestId, tcpuuid, typ, []byte("success")))
	_, werr := w.Write(respVal)
	if werr != nil {
		return werr
	}
	return nil
}

func writeErrorResponse(w http.ResponseWriter, requestId uint16, tcpuuid string, typ string, err error) error {
	respVal, _ := json.Marshal(newJsonErr(requestId, tcpuuid, typ, err))
	_, werr := w.Write(respVal)
	if werr != nil {
		return werr
	}
	return nil
}

func toJsonData(r *http.Request, maxContentLength int64) (*JsonData, error) {
	if r.ContentLength > maxContentLength {
		return nil, errors.New(fmt.Sprintf("Content length too large (%d>%d)", r.ContentLength, maxContentLength))
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	bodyJson := new(JsonData)
	if uerr := json.Unmarshal(body, bodyJson); uerr != nil {
		return nil, uerr
	}

	return bodyJson, nil
}
