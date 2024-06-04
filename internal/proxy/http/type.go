package http

const httpReconnectMsg = "http reconnect"

type JsonData struct {
	// 请求所属 Id
	Id uint16 `json:"id"`
	// 数据所属 tcp 连接的 uuid
	TcpUUid string `json:"tcpUUid"`
	// 数据类型
	Typ string `json:"typ"`
	// 数据内容
	Content []byte `json:"content"`
	// 错误信息
	Err error `json:"err"`
}

func newJsonData(id uint16, tcpUUid string, typ string, content []byte) *JsonData {
	return &JsonData{
		Id:      id,
		TcpUUid: tcpUUid,
		Typ:     typ,
		Content: content,
	}
}

func newJsonErr(id uint16, tcpUUid string, typ string, err error) *JsonData {
	return &JsonData{
		Id:      id,
		TcpUUid: tcpUUid,
		Typ:     typ,
		Err:     err,
	}
}
