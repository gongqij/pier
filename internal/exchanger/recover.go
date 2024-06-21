package exchanger

import (
	"fmt"
	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-kit/hexutil"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/pier/internal/adapt"
	"github.com/meshplus/pier/internal/repo"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

func (ex *Exchanger) handleMissingIBTPByServicePair(begin, end uint64, fromAdapt, toAdapt adapt.Adapt, srcService, targetService string, isReq bool) {
	adaptName := fromAdapt.Name()
	for ; begin <= end; begin++ {
		ex.logger.WithFields(logrus.Fields{
			"service pair": fmt.Sprintf("%s-%s", srcService, targetService),
			"index":        begin,
			"isReq":        isReq,
		}).Info("handle missing event from:" + adaptName)
		ibtp, qerr := ex.queryIBTP(fromAdapt, fmt.Sprintf("%s-%s-%d", srcService, targetService, begin), isReq)
		if qerr != nil {
			ex.logger.Errorf("exchanger stoppped, directly return")
			return
		}
		//transaction timeout rollback in direct mode
		if strings.EqualFold(ex.mode, repo.DirectMode) {
			if isRollback := ex.isIBTPRollbackForDirect(ibtp); isRollback {
				ex.logger.Infof("handle missing meet timeout, do rollbackIBTPForDirect")
				ex.rollbackIBTPForDirect(ibtp)
				return
			}
		}
		ex.logger.Infof("handle missing enter normal logic, do send IBTP[%s] to %s", fmt.Sprintf("%s-%s-%d", srcService, targetService, begin), toAdapt.Name())
		ex.sendIBTP(fromAdapt, toAdapt, ibtp)
		ex.logger.Infof("handle missing meet timeout, end send IBTP[%s] to %s", fmt.Sprintf("%s-%s-%d", srcService, targetService, begin), toAdapt.Name())
	}
}

func (ex *Exchanger) sendIBTP(srcAdapt, destAdapt adapt.Adapt, ibtp *pb.IBTP) {
	adaptName := destAdapt.Name()
	if err := retry.Retry(func(attempt uint) error {
		if err := destAdapt.SendIBTP(ibtp); err != nil {
			ex.logger.Warnf("send IBTP %s to Adapt %s: %s", ibtp.ID(), adaptName, err.Error())
			if err, ok := err.(*adapt.SendIbtpError); ok {
				if err.NeedRetry() {
					select {
					case <-ex.ctx.Done():
						ex.logger.Warningf("exchanger stopped, directly quit retry")
						return nil
					default:
					}
					var qerr error
					ibtp, qerr = ex.queryIBTP(srcAdapt, ibtp.ID(), ibtp.Category() == pb.IBTP_REQUEST)
					if qerr != nil {
						ex.logger.Warningf("exchanger stopped, break sendIBTP retry framework")
						return nil
					}
					return fmt.Errorf("retry sending ibtp")
				}
			}
		}
		return nil
	}, strategy.Backoff(backoff.Fibonacci(500*time.Millisecond))); err != nil {
		ex.logger.Panic(err)
	}
}

func (ex *Exchanger) recover(srcServiceMeta map[string]*pb.Interchain, destServiceMeta map[string]*pb.Interchain) {
	// handle src -> dest
	ex.logger.Info("Start To Recover IBTPs!")
	for _, interchain := range srcServiceMeta {

		ex.logger.Info("[recover] handle srcInterchain --> destInterchain")
		for k, count := range interchain.InterchainCounter {
			destCount, ok := destServiceMeta[interchain.ID].InterchainCounter[k]
			if !ok {
				destCount = 0
			}

			ex.logger.Info("[recover] handle srcInterchain.counter: %d --> destInterchain.counter: %d", count, destCount)

			// handle the situation that dst chain rollback failed but interchainCounter is balanced
			if ex.mode == repo.DirectMode && destCount == count {
				var begin uint64
				for begin = 1; begin <= count; begin++ {
					IBTPid := fmt.Sprintf("%s-%s-%d", interchain.ID, k, begin)
					_, _, txStatus, err := ex.getDirectTransactionMeta(IBTPid)
					if err != nil {
						ex.logger.Errorf("fail to get direct transaction status for ibtp %s, err: %s", IBTPid, err.Error())
						ex.pushErr(err)
						return
					}
					if txStatus == 2 { // transaction status is begin_rollback
						// notify dst chain rollback
						ibtp, qerr := ex.queryIBTP(ex.srcAdapt, IBTPid, true)
						if qerr != nil {
							ex.logger.Errorf("exchanger stopped, interrupt recover")
							ex.pushErr(qerr)
							return
						}
						ibtp.Type = pb.IBTP_RECEIPT_ROLLBACK
						ex.sendIBTPForDirect(ex.srcAdapt, ex.destAdapt, ibtp, ex.isIBTPBelongSrc(ibtp), true)
					}
				}
			}
			// handle unsentIBTP : query IBTP -> sendIBTP
			if destCount < count {
				ex.logger.Info("[recover] enter handle missing")
				ex.handleMissingIBTPByServicePair(destCount+1, count, ex.srcAdapt, ex.destAdapt, interchain.ID, k, true)
				// success then equal index
				destServiceMeta[interchain.ID].InterchainCounter[k] = count
			}
		}
		ex.logger.Info("[recover] handle srcInterchain --> destInterchain finished, start to handle SourceReceiptCounter")
		for k, count := range interchain.SourceReceiptCounter {
			destCount, ok := destServiceMeta[interchain.ID].SourceReceiptCounter[k]
			if !ok {
				destCount = 0
			}
			ex.logger.Info("[recover] handle srcSourceReceiptCounter.counter: %d --> destSourceReceiptCounter.counter: %d", count, destCount)
			// handle unsentIBTP : query IBTP -> sendIBTP
			if destCount < count {
				ex.handleMissingIBTPByServicePair(destCount+1, count, ex.srcAdapt, ex.destAdapt, k, interchain.ID, false)
				destServiceMeta[interchain.ID].SourceReceiptCounter[k] = count
			}
		}
	}
	ex.logger.Info("[recover] handle srcSourceReceiptCounter --> destSourceReceiptCounter finished, start to handle dest->src")
	// handle dest -> src
	for _, interchain := range destServiceMeta {
		for k, count := range interchain.SourceInterchainCounter {
			var srcCount uint64
			if _, ok := srcServiceMeta[interchain.ID]; !ok {
				srcServiceMeta[interchain.ID] = &pb.Interchain{
					InterchainCounter:       make(map[string]uint64, 0),
					ReceiptCounter:          make(map[string]uint64, 0),
					SourceInterchainCounter: make(map[string]uint64, 0),
					SourceReceiptCounter:    make(map[string]uint64, 0),
				}
			} else {
				srcCount, ok = srcServiceMeta[interchain.ID].SourceInterchainCounter[k]
				if !ok {
					srcCount = 0
				}
			}
			ex.logger.Info("[recover] handle destSourceInterchainCounter.counter: %d --> srcSourceInterchainCounter.counter:%d", count, srcCount)
			// srcCount means the count of Interchain from k to interchain.ID
			// handle unsentIBTP : query IBTP -> sendIBTP
			if srcCount < count {
				ex.handleMissingIBTPByServicePair(srcCount+1, count, ex.destAdapt, ex.srcAdapt, k, interchain.ID, true)
				srcServiceMeta[interchain.ID].SourceInterchainCounter[k] = count
			}
		}
		ex.logger.Info("[recover] handle destSourceInterchainCounter --> srcSourceInterchainCounter finished, start to handle receipt part")
		for k, count := range interchain.ReceiptCounter {
			var srcCount uint64
			if _, ok := srcServiceMeta[interchain.ID]; !ok {
				srcServiceMeta[interchain.ID] = &pb.Interchain{
					InterchainCounter:       make(map[string]uint64, 0),
					ReceiptCounter:          make(map[string]uint64, 0),
					SourceInterchainCounter: make(map[string]uint64, 0),
					SourceReceiptCounter:    make(map[string]uint64, 0),
				}
			} else {
				srcCount = srcServiceMeta[interchain.ID].ReceiptCounter[k]
				if !ok {
					srcCount = 0
				}
			}
			ex.logger.Info("[recover] handle destReceiptCounter: %d --> srcReceiptCounter: %d", count, srcCount)
			// handle unsentIBTP : query IBTP -> sendIBTP
			if srcCount < count {
				ex.handleMissingIBTPByServicePair(srcCount+1, count, ex.destAdapt, ex.srcAdapt, interchain.ID, k, false)
				srcServiceMeta[interchain.ID].ReceiptCounter[k] = count
			}
		}
	}

	if ex.mode == repo.RelayMode {
		for serviceID, interchain := range destServiceMeta {
			// deal with source appchain rollback
			for k, interchainCounter := range interchain.InterchainCounter {
				receiptCounter := srcServiceMeta[serviceID].ReceiptCounter[k]

				ex.logger.Infof("check txStatus for service pair %s-%s from %d to %d for rollback", serviceID, k, receiptCounter+1, interchainCounter)
				for i := receiptCounter + 1; i <= interchainCounter; i++ {
					id := fmt.Sprintf("%s-%s-%d", serviceID, k, i)
					ibtp, qerr := ex.queryIBTP(ex.destAdapt, id, true)
					if qerr != nil {
						ex.logger.Errorf("exchanger stopped, interrupt recover")
						return
					}
					bxhProof := &pb.BxhProof{}
					if err := bxhProof.Unmarshal(ibtp.Proof); err != nil {
						ex.logger.Panicf("fail to unmarshal proof %s for ibtp %s", hexutil.Encode(ibtp.Proof), ibtp.ID())
					}

					if bxhProof.TxStatus == pb.TransactionStatus_BEGIN_FAILURE || bxhProof.TxStatus == pb.TransactionStatus_BEGIN_ROLLBACK {
						ex.logger.Infof("ibtp %s txStatus is %v, will rollback", ibtp.ID(), bxhProof.TxStatus)
						ex.sendIBTP(ex.destAdapt, ex.srcAdapt, ibtp)
					}
				}
			}
		}
	}

	ex.logger.Info("End To Recover IBTPs!")
}

func (ex *Exchanger) recoverUnion(srcServiceMeta map[string]*pb.Interchain, destServiceMeta map[string]*pb.Interchain) {
	// handle src -> dest
	ex.logger.Info("Start To Recover IBTPs!")
	for _, interchain := range destServiceMeta {
		for k, count := range interchain.InterchainCounter {
			destCount, ok := srcServiceMeta[interchain.ID].InterchainCounter[k]
			if !ok {
				destCount = 0
			}
			// handle unsentIBTP : query IBTP -> sendIBTP
			var begin = destCount + 1
			ex.handleMissingIBTPByServicePair(begin, count, ex.destAdapt, ex.srcAdapt, interchain.ID, k, true)
			// success then equal index
			srcServiceMeta[interchain.ID].InterchainCounter[k] = count
		}
		for k, count := range interchain.SourceReceiptCounter {
			destCount, ok := srcServiceMeta[interchain.ID].SourceReceiptCounter[k]
			if !ok {
				destCount = 0
			}
			// handle unsentIBTP : query IBTP -> sendIBTP
			var begin = destCount + 1
			ex.handleMissingIBTPByServicePair(begin, count, ex.destAdapt, ex.srcAdapt, k, interchain.ID, false)
			srcServiceMeta[interchain.ID].SourceReceiptCounter[k] = count
		}
	}

	// handle dest -> src
	for _, interchain := range srcServiceMeta {
		for k, count := range interchain.SourceInterchainCounter {
			destCount, ok := destServiceMeta[interchain.ID].SourceInterchainCounter[k]
			if !ok {
				destCount = 0
			}

			// handle unsentIBTP : query IBTP -> sendIBTP
			var begin = destCount + 1
			ex.handleMissingIBTPByServicePair(begin, count, ex.srcAdapt, ex.destAdapt, k, interchain.ID, true)
			destServiceMeta[interchain.ID].SourceInterchainCounter[k] = count
		}

		for k, count := range interchain.ReceiptCounter {
			destCount, ok := destServiceMeta[interchain.ID].ReceiptCounter[k]
			if !ok {
				destCount = 0
			}

			// handle unsentIBTP : query IBTP -> sendIBTP
			var begin = destCount + 1
			ex.handleMissingIBTPByServicePair(begin, count, ex.srcAdapt, ex.destAdapt, interchain.ID, k, false)
			destServiceMeta[interchain.ID].ReceiptCounter[k] = count
		}
	}
	ex.logger.Info("End To Recover IBTPs!")
}
