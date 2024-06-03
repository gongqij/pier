package redisha

import (
	"context"
	"github.com/meshplus/pier/internal/loggers"
	"github.com/meshplus/pier/pkg/rediscli"
	"github.com/sirupsen/logrus"
	"math/rand"
	"strings"
	"time"

	"github.com/meshplus/pier/internal/repo"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// todo: currently outer logic directly New,
// todo: use register later, the constructor param is not proper
//func init() {
//	agency.RegisterPierHAConstructor("redis", New)
//}

//type RedisConfWrapper struct {
//	agency.HAClient
//	Conf repo.Redis
//}

type RedisPierMng struct {
	isMain    chan bool
	ID        string
	LockName  string
	RedisCliW *rediscli.WrapperImpl
	Expire    time.Duration
	Renew     time.Duration
	log       logrus.FieldLogger
	ctx       context.Context
	cancel    context.CancelFunc
}

func (m *RedisPierMng) Start() error {
	m.compete()
	return nil
}

func (m *RedisPierMng) Stop() error {
	m.cancel()
	return m.RedisCliW.Close()
}

func (m *RedisPierMng) IsMain() <-chan bool {
	return m.isMain
}

func (m *RedisPierMng) GetRedisCli() rediscli.Wrapper {
	return m.RedisCliW
}

func New(conf repo.Redis, pierID string) *RedisPierMng {
	ctx, cancel := context.WithCancel(context.Background())
	obj := &RedisPierMng{
		isMain:   make(chan bool),
		ID:       uuid.New().String(),
		LockName: strings.Join([]string{conf.LockPrefix, pierID}, "_"),
		Expire:   time.Duration(conf.LeaseTimeout * int64(time.Second)),
		Renew:    time.Duration(conf.LeaseRenewal * int64(time.Second)),
		log:      loggers.Logger(loggers.App),
		ctx:      ctx,
		cancel:   cancel,
	}
	obj.RedisCliW = rediscli.NewWrapperImpl(
		obj.LockName,
		obj.ID,
		obj.Expire,
		obj.log,
		func() *redis.Client {
			return redis.NewClient(&redis.Options{
				Addr:     conf.Address,
				Password: conf.Password,
				DB:       conf.Database,
			})
		})
	return obj
}

func (m *RedisPierMng) compete() {
	time.Sleep(time.Duration(int64(time.Millisecond) * rand.Int63n(100)))
	locked := m.RedisCliW.Lock()
	if locked {
		m.startMain()
	} else {
		m.startAux()
	}
}

func (m *RedisPierMng) startMain() {
	ticker := time.NewTicker(m.Renew)
	go func() {
		defer ticker.Stop()
		m.isMain <- true
		m.log.Infof("[instance-%s] start in main mod", m.ID)
		for {
			select {
			case <-ticker.C:
				needStartAux := m.RedisCliW.ReExpire() != nil
				if needStartAux {
					m.log.Infof("[instance-%s] quit main mode", m.ID)
					m.startAux()
					return
				}
			case <-m.ctx.Done():
				_ = m.RedisCliW.Unlock()
				return
			}
		}
	}()
}

func (m *RedisPierMng) startAux() {
	ticker := time.NewTicker(m.Expire)
	go func() {
		defer ticker.Stop()
		m.isMain <- false
		m.log.Infof("[instance-%s] start in aux mode", m.ID)
		for {
			select {
			case <-ticker.C:
				locked := m.RedisCliW.Lock()
				if !locked {
					m.log.Infof("[instance-%s] in aux mode, try lock failed", m.ID)
					continue
				}
				m.startMain()
				return
			case <-m.ctx.Done():
				_ = m.RedisCliW.Unlock()
				return
			}
		}
	}()
}
