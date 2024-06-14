package redisha

import (
	"context"
	"github.com/meshplus/pier/internal/loggers"
	"github.com/meshplus/pier/pkg/rediscli"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
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
	conf      repo.Redis
	RedisCliW *rediscli.WrapperImpl
	log       logrus.FieldLogger
	ctx       context.Context
	cancel    context.CancelFunc

	relMasterSignal chan interface{}
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

func (m *RedisPierMng) ReleaseMain() {
	m.relMasterSignal <- struct{}{}
}

func New(conf repo.Redis, pierID string) *RedisPierMng {
	ctx, cancel := context.WithCancel(context.Background())
	obj := &RedisPierMng{
		isMain: make(chan bool),
		ID:     uuid.New().String(),
		conf:   conf,
		log:    loggers.Logger(loggers.App),
		ctx:    ctx,
		cancel: cancel,

		relMasterSignal: make(chan interface{}, 1),
	}
	obj.RedisCliW = rediscli.NewWrapperImpl(
		strings.Join([]string{conf.SendLockPrefix, pierID}, "_"),
		strings.Join([]string{conf.MasterLockPrefix, pierID}, "_"),
		obj.ID,
		int(obj.conf.MasterLeaseTimeout),
		int(obj.conf.SendLeaseTimeout),
		obj.log,
		func() *redis.Client {
			opt := &redis.Options{
				Addr:     conf.Address,
				Password: conf.Password,
				DB:       conf.Database,
			}
			if conf.SelfPort > 0 {
				opt.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
					dialer := &net.Dialer{
						Timeout:   5 * time.Second,
						KeepAlive: 5 * time.Minute,
						LocalAddr: &net.TCPAddr{
							IP:   net.ParseIP("0.0.0.0"),
							Port: conf.SelfPort,
						},
					}
					return dialer.DialContext(ctx, network, addr)
				}
			}
			return redis.NewClient(opt)
		},
	)
	return obj
}

func (m *RedisPierMng) compete() {
	time.Sleep(time.Duration(int64(time.Millisecond) * rand.Int63n(100)))
	locked := m.RedisCliW.MasterLock()
	if locked {
		m.startMain()
	} else {
		m.startAux()
	}
}

func (m *RedisPierMng) startMain() {
	ticker := time.NewTicker(time.Duration(m.conf.MasterLeaseRenewal * int64(time.Second)))
	go func() {
		defer ticker.Stop()
		m.isMain <- true
		m.log.Infof("[instance-%s] start in main mod", m.ID)
		for {
			select {
			case <-ticker.C:
				if !m.RedisCliW.ReNewMaster() {
					m.log.Infof("[instance-%s] quit main mode", m.ID)
					_ = m.RedisCliW.MasterUnlock()
					m.startAux()
					return
				}
			case <-m.relMasterSignal:
				m.log.Infof("[instance-%s] found http connection error, quit main mode", m.ID)
				_ = m.RedisCliW.MasterUnlock()
				m.startAux()
				return
			case <-m.ctx.Done():
				_ = m.RedisCliW.MasterUnlock()
				return
			}
		}
	}()
}

func (m *RedisPierMng) startAux() {
	go func() {
		var timer *time.Timer
		m.isMain <- false
		m.log.Infof("[instance-%s] start in aux mode", m.ID)
		for {
			randDuration := m.conf.MasterLeaseTimeout*1000 - rand.Int63n(1000)
			timer = time.NewTimer(time.Duration(randDuration * int64(time.Millisecond)))
			select {
			case <-timer.C:
				locked := m.RedisCliW.MasterLock()
				if !locked {
					m.log.Infof("[instance-%s] in aux mode, try lock failed", m.ID)
					timer.Stop()
					continue
				}
				m.startMain()
				timer.Stop()
				return
			case <-m.ctx.Done():
				_ = m.RedisCliW.MasterUnlock()
				timer.Stop()
				return
			}
		}
	}()
}
