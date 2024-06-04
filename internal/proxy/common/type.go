package common

//type StreamProcessor[T any] interface {
//	*T
//	FromReader(r io.ReadCloser) error
//	ToReadCloser() io.ReadCloser
//	WaitProcess()
//}

type Data struct {
	// 数据类型
	Typ string

	Content []byte

	// tcp连接的唯一标识
	Uuid string
}

//type DataReadCloser struct {
//	buf *bytes.Buffer
//}
//
//func (d *DataReadCloser) Read(p []byte) (n int, err error) {
//	return d.buf.Read(p)
//}
//
//func (d *DataReadCloser) Close() error {
//	return nil
//}
//
//func (d *Data) ToReadCloser() io.ReadCloser {
//	content, _ := json.Marshal(d)
//	return &DataReadCloser{buf: bytes.NewBuffer(content)}
//}
//
//func (d *Data) FromReader(r io.ReadCloser) error {
//	content, err := io.ReadAll(r)
//	if err != nil {
//		return fmt.Errorf("read file error: %s", err.Error())
//	}
//	_ = r.Close()
//	err = json.Unmarshal(content, d)
//	if err != nil {
//		return fmt.Errorf("parse file to common.Data error: %s", err.Error())
//	}
//	return nil
//}
//
//func (d *Data) WaitProcess() {}
//
//type HttpData struct {
//	Version       uint64
//	ID            string
//	Typ           int // HttpDataType
//	Content       io.ReadCloser
//	ProcessFinish chan interface{}
//}
//
//func (hd *HttpData) ToReadCloser() io.ReadCloser {
//	fixLenContent := make([]byte, 52) // 8+36+8
//	binary.BigEndian.PutUint64(fixLenContent[0:8], hd.Version)
//	copy(fixLenContent[8:44], hd.ID)
//	binary.BigEndian.PutUint64(fixLenContent[44:52], uint64(hd.Typ))
//	return &HttpDataReader{
//		FixLenPart: bytes.NewBuffer(fixLenContent),
//		HttpPart:   hd.Content,
//	}
//}
//
//func (hd *HttpData) FromReader(r io.ReadCloser) error {
//	fixLenContent := make([]byte, 52)
//	n, err := r.Read(fixLenContent)
//	if err != nil {
//		return fmt.Errorf("read fixed length part error: %s, got %d bytes", err.Error(), n)
//	}
//	hd.Version = binary.BigEndian.Uint64(fixLenContent[0:8])
//	idBytes := make([]byte, 36)
//	copy(idBytes, fixLenContent[8:44])
//	hd.ID = string(idBytes)
//	hd.Typ = int(binary.BigEndian.Uint64(fixLenContent[44:52]))
//	hd.Content = r
//	hd.ProcessFinish = make(chan interface{})
//	return nil
//}
//
//func (hd *HttpData) WaitProcess() {
//	<-hd.ProcessFinish
//}
//
//type HttpDataReader struct {
//	FixLenPart io.Reader
//	HttpPart   io.ReadCloser
//}
//
//func (hdr *HttpDataReader) Read(p []byte) (n int, err error) {
//	n, err = hdr.FixLenPart.Read(p)
//	if err == io.EOF {
//		return hdr.HttpPart.Read(p)
//	}
//	return
//}
//
//func (hdr *HttpDataReader) Close() error {
//	return hdr.HttpPart.Close()
//}
