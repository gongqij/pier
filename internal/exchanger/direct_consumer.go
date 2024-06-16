package exchanger

import (
	"fmt"
	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/pier/internal/adapt"
	"github.com/meshplus/pier/internal/adapt/appchain_adapter"
	"github.com/sirupsen/logrus"
	"time"
)

func (ex *Exchanger) listenIBTPFromDestAdaptForDirect(servicePair string) {
	defer ex.wg.Done()
	for {
		select {
		case <-ex.ctx.Done():
			ex.logger.Info("ListenIBTPFromDestAdapt Stop!")
			return
		case ibtp, ok := <-ex.destIBTPMap[servicePair]:
			if !ok {
				ex.logger.Warn("Unexpected closed channel while listening on interchain ibtp")
				return
			}
			ex.logger.WithFields(logrus.Fields{"index": ibtp.Index, "type": ibtp.Type, "ibtp_id": ibtp.ID()}).Info("Receive ibtp from :", ex.destAdaptName)
			index := ex.getCurrentIndexFromDest(ibtp)
			ex.logger.Infof("current index from destAdaptor for ibtp: %s is %d", ibtp.ID(), index)

			if index >= ibtp.Index {
				if ibtp.Type == pb.IBTP_RECEIPT_ROLLBACK {
					// dst chain receive rollback, need rollback
					ex.sendIBTPForDirect(ex.destAdapt, ex.srcAdapt, ibtp, !ex.isIBTPBelongSrc(ibtp), true)
				} else if ibtp.Type == pb.IBTP_RECEIPT_ROLLBACK_END {
					// receive rollback end from dst chain, src chain change transaction status
					ex.sendIBTPForDirect(ex.destAdapt, ex.srcAdapt, ibtp, !ex.isIBTPBelongSrc(ibtp), false)
				} else {
					ex.logger.WithFields(logrus.Fields{"index": ibtp.Index, "to_counter": index, "ibtp_id": ibtp.ID()}).Info("Ignore ibtp")
				}
				continue
			}

			if index+1 < ibtp.Index {
				ex.logger.WithFields(logrus.Fields{"index": ibtp.Index, "to": ibtp.To}).Info("Get missing ibtp")
				ex.handleMissingIBTPByServicePair(index+1, ibtp.Index-1, ex.destAdapt, ex.srcAdapt, ibtp.From, ibtp.To, !ex.isIBTPBelongSrc(ibtp))
			}

			if isRollback := ex.isIBTPRollbackForDirect(ibtp); isRollback {
				// receipt time out, need update ibtp with invoke info
				ibtp, _ = ex.srcAdapt.QueryIBTP(ibtp.ID(), true)
				ex.rollbackIBTPForDirect(ibtp)
			} else {
				ex.sendIBTPForDirect(ex.destAdapt, ex.srcAdapt, ibtp, !ex.isIBTPBelongSrc(ibtp), false)
			}
			ex.logger.Infof("[direct-consumer] send IBTP [%s] returned", ibtp.ID())

			if ex.isIBTPBelongSrc(ibtp) {
				ex.destServiceMeta[ibtp.From].ReceiptCounter[ibtp.To] = ibtp.Index
			} else {
				ex.destServiceMeta[ibtp.To].SourceInterchainCounter[ibtp.From] = ibtp.Index
			}
		}
	}
}

func (ex *Exchanger) listenIBTPFromSrcAdaptForDirect(servicePair string) {
	defer ex.wg.Done()
	for {
		select {
		case <-ex.ctx.Done():
			ex.logger.Info("ListenIBTPFromSrcAdapt Stop!")
			return
		case ibtp, ok := <-ex.srcIBTPMap[servicePair]:
			if !ok {
				ex.logger.Warn("Unexpected closed channel while listening on interchain ibtp")
				return
			}
			ex.logger.WithFields(logrus.Fields{"index": ibtp.Index, "type": ibtp.Type, "ibtp_id": ibtp.ID()}).Info("Receive ibtp from :", ex.srcAdaptName)
			index := ex.getCurrentIndexFromSrc(ibtp)
			if index >= ibtp.Index {
				if ibtp.Type == pb.IBTP_RECEIPT_ROLLBACK_END {
					ex.sendIBTPForDirect(ex.srcAdapt, ex.destAdapt, ibtp, ex.isIBTPBelongSrc(ibtp), false)
				} else {
					ex.logger.WithFields(logrus.Fields{"index": ibtp.Index, "to_counter": index, "ibtp_id": ibtp.ID()}).Info("Ignore ibtp")
				}
				continue
			}

			if index+1 < ibtp.Index {
				ex.logger.WithFields(logrus.Fields{"index": ibtp.Index, "to": ibtp.To}).Info("Get missing ibtp")
				ex.handleMissingIBTPByServicePair(index+1, ibtp.Index-1, ex.srcAdapt, ex.destAdapt, ibtp.From, ibtp.To, ex.isIBTPBelongSrc(ibtp))
			}

			if isRollback := ex.isIBTPRollbackForDirect(ibtp); isRollback {
				ex.rollbackIBTPForDirect(ibtp)
			} else {
				ex.sendIBTPForDirect(ex.srcAdapt, ex.destAdapt, ibtp, ex.isIBTPBelongSrc(ibtp), false)
			}

			if ex.isIBTPBelongSrc(ibtp) {
				ex.srcServiceMeta[ibtp.From].InterchainCounter[ibtp.To] = ibtp.Index
			} else {
				ex.srcServiceMeta[ibtp.To].SourceReceiptCounter[ibtp.From] = ibtp.Index
			}
		}
	}
}

func (ex *Exchanger) isIBTPRollbackForDirect(ibtp *pb.IBTP) bool {
	if !ex.isIBTPBelongSrc(ibtp) || ibtp.Type == pb.IBTP_RECEIPT_ROLLBACK || ibtp.Type == pb.IBTP_RECEIPT_ROLLBACK_END {
		return false
	}

	startTimeStamp, timeoutPeriod, txStatus, err := ex.srcAdapt.(*appchain_adapter.AppchainAdapter).GetDirectTransactionMeta(ibtp.ID())
	if err != nil {
		ex.logger.Errorf("get transaction meta with %s", ibtp.ID(), "error", err.Error())
	} else {
		ex.logger.Infof("[isIBTPRollbackForDirect] ibtp.Type: %s, startTimeStamp: %d, timeoutPeriod: %d, "+
			"txStatus: %v, for IBTP [%s], time.Now().Unix(): %d", ibtp.Type.String(), startTimeStamp, timeoutPeriod,
			txStatus, ibtp.ID(), time.Now().Unix())
		if uint64(time.Now().Unix()) > startTimeStamp {
			return uint64(time.Now().Unix())-startTimeStamp > timeoutPeriod
		}
	}
	return false
}

func (ex *Exchanger) rollbackIBTPForDirect(ibtp *pb.IBTP) {
	ibtp.Type = pb.IBTP_RECEIPT_ROLLBACK

	// src chain rollback
	ex.logger.Infof("src chain start rollback %s", ibtp.ID())
	ex.sendIBTPForDirect(ex.srcAdapt, ex.srcAdapt, ibtp, ex.isIBTPBelongSrc(ibtp), true)
	ex.logger.Infof("src chain rollback %s end", ibtp.ID())

	// src chain rollback end, notify dst chain rollback
	ex.logger.Infof("notify dst chain rollback")
	ex.sendIBTPForDirect(ex.srcAdapt, ex.destAdapt, ibtp, ex.isIBTPBelongSrc(ibtp), true)
	ex.logger.Infof("dst chain rollback %s end", ibtp.ID())
}

func (ex *Exchanger) sendIBTPForDirect(fromAdapt, toAdapt adapt.Adapt, ibtp *pb.IBTP, isReq bool, isRollback bool) {
	if err := retry.Retry(func(attempt uint) error {
		ex.logger.Infof("start sendIBTP to Adapt:%s", toAdapt.Name())
		if err := toAdapt.SendIBTP(ibtp); err != nil {
			// if err occurs, try to get new ibtp and resend
			if err, ok := err.(*adapt.SendIbtpError); ok {
				if err.NeedRetry() {
					select {
					case <-ex.ctx.Done():
						ex.logger.Warningf("exchanger stopped, directly quit retry")
						return nil
					default:
					}
					ex.logger.Errorf("send IBTP to Adapt:%s", ex.destAdaptName, "error", err.Error())
					// query to new ibtp
					var qerr error
					ibtp, qerr = ex.queryIBTP(fromAdapt, ibtp.ID(), isReq)
					if qerr != nil {
						ex.logger.Warningf("exchanger stopped, break SendIBTP retry framework")
						return nil
					}
					// set ibtp rollback
					if isRollback {
						ibtp.Type = pb.IBTP_RECEIPT_ROLLBACK
					}
					// check if retry timeout
					if isTimeout := ex.isIBTPRollbackForDirect(ibtp); isTimeout {
						ex.rollbackIBTPForDirect(ibtp)
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
