package rediscli

import (
	"context"
	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"sync"
)

const (
	sendLockVal = "placeholder"
)

type WrapperImpl struct {
	sendLockName string
	sendLockVal  string
	sendExpire   int

	masterLockName string
	masterLockVal  string
	masterExpire   int
	log            logrus.FieldLogger

	cli *redis.Client

	ctx    context.Context
	cancel context.CancelFunc

	closed    bool
	closeLock *sync.RWMutex
}

func NewWrapperImpl(
	sendLockName string,
	masterLockName string,
	masterLockVal string,
	masterExpire int,
	sendExpire int,
	log logrus.FieldLogger,
	newCliFunc func() *redis.Client) *WrapperImpl {
	ctx, cancel := context.WithCancel(context.Background())
	return &WrapperImpl{
		sendLockName:   sendLockName,
		sendLockVal:    sendLockVal,
		sendExpire:     sendExpire,
		masterLockName: masterLockName,
		masterLockVal:  masterLockVal,
		masterExpire:   masterExpire,
		log:            log,
		cli:            newCliFunc(),
		closeLock:      &sync.RWMutex{},
		ctx:            ctx,
		cancel:         cancel,
		closed:         false,
	}
}

func (w *WrapperImpl) Close() error {
	w.closeLock.Lock()
	defer w.closeLock.Unlock()
	w.closed = true
	w.cancel()
	return w.cli.Close()
}

func (w *WrapperImpl) MasterLock() bool {
	w.closeLock.RLock()
	defer w.closeLock.RUnlock()
	if w.closed {
		w.log.Error(ErrClosed.Error())
		return false
	}
	var locked bool
	err := retry.Retry(func(attempt uint) error {
		var lerr error
		locked, lerr = w.masterLock()
		if lerr != nil {
			return lerr
		}
		return nil
	}, strategy.Limit(5))
	if err != nil {
		w.log.Errorf("[RedisCliObj] try masterLock failed with error: %s, retry 5 times still failed", err.Error())
	}
	return locked
}

func (w *WrapperImpl) MasterUnlock() bool {
	w.closeLock.RLock()
	defer w.closeLock.RUnlock()
	if w.closed {
		w.log.Error(ErrClosed.Error())
		return false
	}
	var unlocked bool
	err := retry.Retry(func(attempt uint) error {
		var lerr error
		unlocked, lerr = w.masterUnlock()
		if lerr != nil {
			return lerr
		}
		return nil
	}, strategy.Limit(5))
	if err != nil {
		w.log.Errorf("[RedisCliObj] try masterUnlock failed with error: %s, retry 5 times still failed", err.Error())
	}
	return unlocked
}

func (w *WrapperImpl) SendLock() bool {
	w.closeLock.RLock()
	defer w.closeLock.RUnlock()
	if w.closed {
		return false
	}
	var locked bool
	err := retry.Retry(
		func(attempt uint) error {
			var lerr error
			locked, lerr = w.sendLock()
			if lerr != nil {
				return lerr
			}
			return nil
		},
		strategy.Limit(3),
	)
	if err != nil {
		w.log.Errorf("[RedisCliObj] try SendLock failed with error: %s, retry 3 times still failed", err.Error())
	}
	return locked
}

func (w *WrapperImpl) SendUnlock() bool {
	w.closeLock.RLock()
	defer w.closeLock.RUnlock()
	if w.closed {
		w.log.Error(ErrClosed.Error())
		return false
	}
	var unlocked bool
	err := retry.Retry(func(attempt uint) error {
		var lerr error
		unlocked, lerr = w.sendUnlock()
		if lerr != nil {
			return lerr
		}
		return nil
	}, strategy.Limit(3))
	if err != nil {
		w.log.Errorf("[RedisCliObj] try sendUnlock failed with error: %s, retry 3 times still failed", err.Error())
	}
	return unlocked
}

func (w *WrapperImpl) ReNewMaster() bool {
	w.closeLock.RLock()
	defer w.closeLock.RUnlock()
	if w.closed {
		w.log.Error(ErrClosed.Error())
		return false
	}
	var success bool
	err := retry.Retry(func(attempt uint) error {
		var eerr error
		success, eerr = w.reNewMaster()
		if eerr != nil {
			return eerr
		}
		return nil
	}, strategy.Limit(5))
	if err != nil {
		w.log.Errorf("[RedisCliObj] try renewMaster failed with error: %s, retry 5 times still failed", err.Error())
	}
	return success
}

// ==================== inner function ====================

func (w *WrapperImpl) sendLock() (bool, error) {
	script := redis.NewScript(lockSend)
	res, err := script.Run(w.ctx, w.cli, []string{w.masterLockName, w.sendLockName}, w.masterLockVal, w.sendLockVal, w.sendExpire).Int64()
	if err != nil {
		w.log.Errorf("[RedisCliObj] sendLock with key[%s] value [%s] error: %s", w.sendLockName, w.sendLockVal, err.Error())
		return false, err
	}
	if res != 1 {
		w.log.Infof("[RedisCliObj] sendLock with key[%s] value [%s] error: %d", w.sendLockName, w.sendLockVal, res)
		return false, nil
	}
	return true, nil
}

func (w *WrapperImpl) sendUnlock() (bool, error) {
	script := redis.NewScript(unlockSend)
	res, err := script.Run(w.ctx, w.cli, []string{w.masterLockName, w.sendLockName}, w.masterLockVal).Int64()
	if err != nil {
		w.log.Errorf("[RedisCliObj] sendUnlock with key[%s] value [%s] error: %s", w.masterLockName, w.masterLockVal, err.Error())
		return false, err
	}
	if res != 1 {
		w.log.Infof("[RedisCliObj] sendUnlock with key[%s] value [%s] error: %s", w.masterLockName, w.masterLockVal, err.Error())
		return false, nil
	}
	return true, nil
}

func (w *WrapperImpl) masterLock() (bool, error) {
	script := redis.NewScript(lockMaster)
	res, err := script.Run(w.ctx, w.cli, []string{w.masterLockName, w.sendLockName}, w.masterLockVal, w.masterExpire).Int64()
	if err != nil {
		w.log.Errorf("[RedisCliObj] masterLock with key[%s] value [%s] error: %s", w.masterLockName, w.masterLockVal, err.Error())
		return false, err
	}
	if res != 1 {
		w.log.Infof("[RedisCliObj] masterLock with key[%s] value [%s] has result %d", w.masterLockName, w.masterLockVal, res)
		return false, nil
	}
	return true, nil
}

func (w *WrapperImpl) masterUnlock() (bool, error) {
	script := redis.NewScript(unlockMaster)
	res, err := script.Run(w.ctx, w.cli, []string{w.masterLockName}, w.masterLockVal).Int64()
	if err != nil {
		w.log.Errorf("[RedisCliObj] masterUnlock with key[%s] value [%s] error: %s", w.masterLockName, w.masterLockVal, err.Error())
		return false, err
	}
	if res != 1 {
		w.log.Infof("[RedisCliObj] masterUnlock with key[%s] value [%s] with result: %d", w.masterLockName, w.masterLockVal, res)
		return false, nil
	}
	return true, nil
}

func (w *WrapperImpl) reNewMaster() (bool, error) {
	script := redis.NewScript(expireMaster)
	res, err := script.Run(w.ctx, w.cli, []string{w.masterLockName}, w.masterLockVal, w.masterExpire).Int64()
	if err != nil {
		w.log.Errorf("re-masterExpire with key[%s] value [%s] error: %s", w.masterLockName, w.masterLockVal, err.Error())
		return false, err
	}
	if res != 1 {
		w.log.Infof("re-masterExpire with key[%s] value [%s] result: %d", w.masterLockName, w.masterLockVal, res)
		return false, nil
	}
	return true, nil
}

const (
	lockSend = `
		if(redis.call('get',KEYS[1])==ARGV[1] and redis.call('exists',KEYS[2])==0) then
			local result = redis.call('SET', KEYS[2], ARGV[2], 'NX', 'EX', tonumber(ARGV[3]))
			if result then
				return 1
			else
				return 0
			end
		else
			return 2
		end
	`
	unlockSend = `
		if(redis.call('get',KEYS[1])==ARGV[1]) then
			return redis.call('DEL',KEYS[2])
		else
			return 2
		end
	`
	lockMaster = `
		if(redis.call('exists',KEYS[1])==0 and redis.call('exists',KEYS[2])==0) then
			local result = redis.call('SET', KEYS[1], ARGV[1], 'NX', 'EX', tonumber(ARGV[2]))
			if result then
				return 1
			else
				return 0
			end
		else
			return 2
		end
	`
	unlockMaster = `
		if(redis.call('get',KEYS[1])==ARGV[1]) then
			return redis.call('DEL',KEYS[1])
		else
			return 2
		end
	`
	expireMaster = `
		if(redis.call('get',KEYS[1])==ARGV[1]) then
			return redis.call('EXPIRE', KEYS[1], tonumber(ARGV[2]))
		else
			return 2
		end
	`
)
