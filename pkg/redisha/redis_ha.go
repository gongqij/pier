package redisha

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/pier/internal/repo"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
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
	isMain   chan bool
	ID       string
	LockName string
	RedisCli *redis.Client
	Expire   time.Duration
	Renew    time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

func (m *RedisPierMng) Start() error {
	m.compete()
	return nil
}

func (m *RedisPierMng) Stop() error {
	m.cancel()
	return nil
}

func (m *RedisPierMng) IsMain() <-chan bool {
	return m.isMain
}

func New(conf repo.Redis, pierID string) agency.PierHA {
	ctx, cancel := context.WithCancel(context.Background())
	obj := &RedisPierMng{
		isMain:   make(chan bool),
		ID:       uuid.New().String(),
		LockName: strings.Join([]string{conf.LockPrefix, pierID}, "_"),
		RedisCli: redis.NewClient(&redis.Options{
			Addr:     conf.Address,
			Password: conf.Password,
			DB:       conf.Database,
		}),
		Expire: time.Duration(conf.LeaseTimeout * int64(time.Second)),
		Renew:  time.Duration(conf.LeaseRenewal * int64(time.Second)),
		ctx:    ctx,
		cancel: cancel,
	}
	return obj
}

func (m *RedisPierMng) compete() {
	time.Sleep(time.Duration(int64(time.Millisecond) * rand.Int63n(1000)))
	_ = retry.Retry(func(attempt uint) error {
		locked, err := m.lock()
		if err != nil {
			fmt.Printf("[instance-%s] compete lock error: %s\n", m.ID, err.Error())
			return err
		}
		if locked {
			m.startMain()
		} else {
			m.startAux()
		}
		return nil
	}, strategy.Wait(2*time.Second))
}

func (m *RedisPierMng) startMain() {
	ticker := time.NewTicker(m.Renew)
	go func() {
		m.isMain <- true
		fmt.Printf("[instance-%s] start in main mode\n", m.ID)
		for {
			select {
			case <-ticker.C:
				needStartAux := true
				str, err := m.RedisCli.Get(m.ctx, m.LockName).Result()
				if err != nil {
					fmt.Printf("[instance-%s] get lock value error: %s\n", m.ID, err.Error())
				} else {
					if str == m.ID {
						if rerr := retry.Retry(
							func(attempt uint) error {
								success, eerr := m.RedisCli.Expire(m.ctx, m.LockName, m.Expire).Result()
								if eerr != nil {
									fmt.Printf("[instance-%s] set expire error: %s\n", m.ID, eerr.Error())
									return eerr
								}
								if success {
									// value == self.ID && set Expire success, stay main
									needStartAux = false
								}
								return nil
							},
							strategy.Limit(3),
						); rerr != nil {
							fmt.Printf("[instance-%s] retry set expire finished with error: %s\n", m.ID, rerr.Error())
						}
					}
				}
				if needStartAux {
					fmt.Printf("[instance-%s] quit main mode\n", m.ID)
					m.startAux()
					return
				}
			case <-m.ctx.Done():
				_ = m.unlock()
				return
			}
		}
	}()
}

func (m *RedisPierMng) startAux() {
	ticker := time.NewTicker(m.Expire)
	go func() {
		m.isMain <- false
		fmt.Printf("[instance-%s] start in aux mode\n", m.ID)
		for {
			select {
			case <-ticker.C:
				locked, err := m.lock()
				if err != nil {
					fmt.Printf("[instance-%s] in aux mode, try lock error: %s\n", m.ID, err.Error())
					continue
				}
				if !locked {
					fmt.Printf("[instance-%s] in aux mode, try lock failed\n", m.ID)
					continue
				}
				m.startMain()
				return
			case <-m.ctx.Done():
				_ = m.unlock()
				return
			}
		}
	}()
}

func (m *RedisPierMng) lock() (bool, error) {
	return m.RedisCli.SetNX(m.ctx, m.LockName, m.ID, m.Expire).Result()
}

func (m *RedisPierMng) unlock() error {
	script := redis.NewScript(LauCheckAndDelete)
	res, err := script.Run(m.ctx, m.RedisCli, []string{m.LockName}, m.ID).Int64()
	if err != nil {
		fmt.Printf("[instance-%s] unlock error: %s\n", m.ID, err.Error())
		return err
	}
	if res != 1 {
		fmt.Printf("[instance-%s] unlock failed: res %d\n", m.ID, res)
		return errors.New("can not unlock because del result not is one")
	}
	return nil
}

const (
	LauCheckAndDelete = `
		if(redis.call('get',KEYS[1])==ARGV[1]) then
		return redis.call('del',KEYS[1])
		else
		return 0
		end
	`
)
