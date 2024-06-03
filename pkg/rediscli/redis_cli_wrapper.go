package rediscli

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type WrapperImpl struct {
	lockName string
	lockVal  string
	expire   time.Duration
	log      logrus.FieldLogger

	cli *redis.Client

	ctx    context.Context
	cancel context.CancelFunc

	closed    bool
	closeLock *sync.RWMutex
}

func NewWrapperImpl(
	lockName string,
	lockVal string,
	expire time.Duration,
	log logrus.FieldLogger,
	newCliFunc func() *redis.Client) *WrapperImpl {
	ctx, cancel := context.WithCancel(context.Background())
	return &WrapperImpl{
		lockName:  lockName,
		lockVal:   lockVal,
		expire:    expire,
		log:       log,
		cli:       newCliFunc(),
		closeLock: &sync.RWMutex{},
		ctx:       ctx,
		cancel:    cancel,
		closed:    false,
	}
}

func (w *WrapperImpl) Close() error {
	w.closeLock.Lock()
	defer w.closeLock.Unlock()
	w.closed = true
	w.cancel()
	return w.cli.Close()
}

func (w *WrapperImpl) Lock() bool {
	w.closeLock.RLock()
	defer w.closeLock.RUnlock()
	if w.closed {
		return false
	}
	var locked bool
	err := retry.Retry(func(attempt uint) error {
		var lerr error
		locked, lerr = w.lock()
		if lerr != nil {
			w.log.Infof("set key[%s] val[%s] compete lock error: %s", w.lockName, w.lockVal, lerr.Error())
			return lerr
		}
		return nil
	}, strategy.Limit(5))
	if err != nil {
		w.log.Errorf("[RedisCliObj] try lock failed with error: %s, spin retry till success", err.Error())
	}
	return locked
}

func (w *WrapperImpl) Unlock() error {
	w.closeLock.RLock()
	defer w.closeLock.RUnlock()
	if w.closed {
		return ErrClosed
	}
	err := retry.Retry(func(attempt uint) error {
		return w.unlock()
	}, strategy.Limit(5))
	if err != nil {
		w.log.Errorf("[RedisCliObj] try lock failed with error: %s, spin retry till success", err.Error())
	}
	return err
}

func (w *WrapperImpl) HeldLock() bool {
	w.closeLock.RLock()
	defer w.closeLock.RUnlock()
	if w.closed {
		return false
	}
	var held bool
	val, err := w.get()
	if err == nil && val == w.lockVal {
		held = true
	}
	if err != nil {
		logrus.Warningf("[RedisCliObj] get lock failed with error: %s", err.Error())
	}
	return held
}

func (w *WrapperImpl) ReExpire() error {
	w.closeLock.RLock()
	defer w.closeLock.RUnlock()
	if w.closed {
		return ErrClosed
	}
	err := retry.Retry(
		func(attempt uint) error {
			return w.reExpire()
		},
		strategy.Limit(5),
	)
	if err != nil {
		w.log.Warnf("retry set expire for key[%s] value[%s] finished with error: %s\n",
			w.lockName, w.lockVal, err.Error())
	}
	return err
}

// ==================== inner function ====================

func (w *WrapperImpl) lock() (bool, error) {
	return w.cli.SetNX(w.ctx, w.lockName, w.lockVal, w.expire).Result()
}

func (w *WrapperImpl) unlock() error {
	script := redis.NewScript(checkAndDelete)
	res, err := script.Run(w.ctx, w.cli, []string{w.lockName}, w.lockVal).Int64()
	if err != nil {
		w.log.Errorf("unlock with key[%s] value [%s] error: %s", w.lockName, w.lockVal, err.Error())
		return err
	}
	if res != 1 {
		w.log.Errorf("unlock with key[%s] value [%s] error: %s", w.lockName, w.lockVal, err.Error())
		return errors.New("can not unlock because del result not is one")
	}
	return nil
}

func (w *WrapperImpl) get() (string, error) {
	return w.cli.Get(w.ctx, w.lockName).Result()
}

func (w *WrapperImpl) reExpire() error {
	script := redis.NewScript(checkAndExpire)
	res, err := script.Run(w.ctx, w.cli, []string{w.lockName}, w.lockVal, int(w.expire.Seconds())).Int64()
	if err != nil {
		w.log.Errorf("re-expire with key[%s] value [%s] error: %s", w.lockName, w.lockVal, err.Error())
		return err
	}
	if res != 1 {
		w.log.Errorf("re-expire with key[%s] value [%s] error: %s", w.lockName, w.lockVal, err.Error())
		return errors.New("can not re-expire because re-expire result not is one")
	}
	return nil
}

const (
	checkAndDelete = `
		if(redis.call('get',KEYS[1])==ARGV[1]) then
		return redis.call('del',KEYS[1])
		else
		return 0
		end
	`
	checkAndExpire = `
		if(redis.call('get',KEYS[1])==ARGV[1]) then
		return redis.call('expire', key, tonumber(ARGV[2]))
		else
		return 0
		end
	`
)
