package executor

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	rpcx "github.com/meshplus/go-bitxhub-client"
	"github.com/meshplus/pier/internal/txcrypto/mock_txcrypto"
	"github.com/meshplus/pier/pkg/plugins/mock_client"
	"github.com/stretchr/testify/require"
)

const (
	from = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	to   = "0x0915fdfc96232c95fb9c62d27cc9dc0f13f50161"
)

func TestExecute(t *testing.T) {
	exec, cli := prepare(t)
	defer exec.storage.Close()

	// set expect values
	ibtp1 := getIBTP(t, uint64(1), pb.IBTP_INTERCHAIN)
	ibtp1Receipt := getIBTPReceipt(t, uint64(1), pb.IBTP_RECEIPT_SUCCESS)
	ibtp2 := getIBTP(t, uint64(2), pb.IBTP_INTERCHAIN)
	ibtp2Receipt := getIBTPReceipt(t, uint64(2), pb.IBTP_RECEIPT_FAILURE)
	ret1 := &pb.SubmitIBTPResponse{
		Status: true,
		Result: ibtp1Receipt,
	}
	ret2 := &pb.SubmitIBTPResponse{
		Status: false,
		Result: ibtp2Receipt,
	}
	//ag.EXPECT().SendIBTP(gomock.Any()).Return(getReceipt(), nil).AnyTimes()
	//ag.EXPECT().GetIBTPByID(gomock.Any()).Return(getIBTP(t, 2, pb.IBTP_INTERCHAIN), nil).Times(1)
	cli.EXPECT().SubmitIBTP(ibtp1).Return(ret1, nil).AnyTimes()
	cli.EXPECT().SubmitIBTP(ibtp2).Return(ret2, nil).AnyTimes()
	cli.EXPECT().SubmitIBTP(ibtp1Receipt).Return(ret1, nil).AnyTimes()
	cli.EXPECT().SubmitIBTP(ibtp2Receipt).Return(ret2, nil).AnyTimes()
	cli.EXPECT().Stop().Return(nil).AnyTimes()

	// start executor
	require.Nil(t, exec.Start())

	// test for normal ibtp execution
	receipt, err := exec.ExecuteIBTP(ibtp1)
	require.Nil(t, err)
	require.NotNil(t, receipt)
	receipt, err = exec.ExecuteIBTP(ibtp1Receipt)
	require.Nil(t, err)
	require.Nil(t, receipt)

	// test for ibtp execute failure
	receipt, err = exec.ExecuteIBTP(ibtp2)
	require.Nil(t, err)
	require.NotNil(t, receipt)

	receipt, err = exec.ExecuteIBTP(ibtp2Receipt)
	require.Nil(t, err)
	require.Nil(t, receipt)

	time.Sleep(500 * time.Microsecond)
	require.Nil(t, exec.Stop())
}

func TestQueryReceipt(t *testing.T) {
	exec, cli := prepare(t)
	defer exec.storage.Close()

	originalIBTP := getIBTP(t, 1, pb.IBTP_INTERCHAIN)
	args := [][]byte{[]byte("Alice"), []byte("100")}

	cli.EXPECT().GetInMessage(from, uint64(1)).Return(args, nil).AnyTimes()

	receipt, err := exec.QueryIBTPReceipt(from, 1, originalIBTP)
	require.Nil(t, err)

	require.Equal(t, originalIBTP.From, receipt.From)
	require.Equal(t, originalIBTP.To, receipt.To)
	require.Equal(t, originalIBTP.Index, receipt.Index)

	receiptPd := &pb.Payload{}
	require.Nil(t, receiptPd.Unmarshal(receipt.Payload))

	receiptContent := &pb.Content{}
	require.Nil(t, receiptContent.Unmarshal(receiptPd.Content))

	originalPd := &pb.Payload{}
	require.Nil(t, originalPd.Unmarshal(originalIBTP.Payload))

	originalContent := &pb.Content{}
	require.Nil(t, originalContent.Unmarshal(originalPd.Content))

	require.Equal(t, receiptContent.Func, originalContent.Callback)
	require.Equal(t, receiptContent.Args[1:], args)
}

func prepare(t *testing.T) (*ChannelExecutor, *mock_client.MockClient) {
	mockCtl := gomock.NewController(t)
	mockCtl.Finish()
	cli := mock_client.NewMockClient(mockCtl)
	cryptor := mock_txcrypto.NewMockCryptor(mockCtl)

	tmpDir, err := ioutil.TempDir("", "storage")
	require.Nil(t, err)
	storage, err := leveldb.New(tmpDir)
	require.Nil(t, err)
	meta := &rpcx.Interchain{
		ID: from,
	}

	cli.EXPECT().GetInMeta().Return(make(map[string]uint64), nil).AnyTimes()
	cli.EXPECT().GetCallbackMeta().Return(make(map[string]uint64), nil).AnyTimes()
	exec, err := New(cli, meta.ID, storage, cryptor, log.NewWithModule("executor"))
	require.Nil(t, err)
	return exec, cli
}

func getReceipt() *pb.Receipt {
	return &pb.Receipt{
		Version: []byte("0.4.5"),
		TxHash:  &types.Hash{},
		Ret:     nil,
		Status:  0,
	}
}

func getIBTPReceipt(t *testing.T, index uint64, typ pb.IBTP_Type) *pb.IBTP {
	receipt := getIBTP(t, index, typ)
	receipt.From, receipt.To = receipt.To, receipt.From
	return receipt
}

func getIBTP(t *testing.T, index uint64, typ pb.IBTP_Type) *pb.IBTP {
	ct := &pb.Content{
		SrcContractId: from,
		DstContractId: to,
		Func:          "set",
		Args:          [][]byte{[]byte("Alice")},
		Callback:      "interchainConfirm",
	}
	c, err := ct.Marshal()
	require.Nil(t, err)

	pd := pb.Payload{
		Encrypted: false,
		Content:   c,
	}
	ibtppd, err := pd.Marshal()
	require.Nil(t, err)

	return &pb.IBTP{
		From:      from,
		To:        to,
		Payload:   ibtppd,
		Index:     index,
		Type:      typ,
		Timestamp: time.Now().UnixNano(),
	}
}
