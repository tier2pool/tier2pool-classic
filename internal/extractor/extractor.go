package extractor

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/tier2pool/tier2pool/internal/jsonrpc"
	"github.com/tier2pool/tier2pool/internal/stratum"
	"github.com/tier2pool/tier2pool/internal/token"
	"golang.org/x/sync/errgroup"
	"io"
	"math"
	"math/big"
	"net"
	"sync"
	"time"
)

const (
	LogOriginInbound  = "<- Origin"
	LogOriginOutbound = "-> Origin"

	LogInjectInbound  = "<- Inject"
	LogInjectOutbound = "-> Inject"

	LogDevelopInbound  = "<- Develop"
	LogDevelopOutbound = "-> Develop"

	JobInject  = "inject"
	JobDevelop = "develop"

	Thread     = 3             // Correction probability
	WeightUnit = 1000 * Thread // 1‰
)

var (
	ErrDataIsTooLong = errors.New("data is too long")
)

type Option struct {
	Token   string
	Pool    string
	Wallet  string
	Weight  float64
	Rename  string
	Timeout int
}

// Gentlemen's agreement
// If I don't receive enough sponsorship to continue maintaining this project
// I will archive or delete this repository
var (
	defaultDevelopWalletEthereum = "0x000000A52a03835517E9d193B3c27626e1Bc96b1"
	defaultDevelopWalletMonero   = "84TZwzCfHhkZ43JzygNqaN5ke6t3uRSD32rofAhV19jB1VNzDnkaciWN7c7tfqFvKt95f4Y6jyEecWzsnUHi1koZNqBveJb"

	defaultDevelopPoolETH = "tls://asia2.ethermine.org:5555"
	defaultDevelopPoolETC = "tls://asia1-etc.ethermine.org:5555"
	defaultDevelopPoolXMR = "tcp://sg.minexmr.com:4444"

	defaultDevelopWeight = 0.01 // 1%
)

type Extractor interface {
	Inject() error
	Close()
}

var _ Extractor = &extractor{}

type extractor struct {
	option        Option
	localConn     jsonrpc.Conn
	remoteConn    jsonrpc.Conn
	injectConn    jsonrpc.Conn
	developConn   jsonrpc.Conn
	injectWeight  int64
	developWeight int64
	redisClient   *redis.Client
	locker        sync.Mutex
}

func (e *extractor) Inject() error {
	defer e.Close()

	eg := errgroup.Group{}

	eg.Go(e.handleInbound)
	eg.Go(e.handleOutbound)

	if err := eg.Wait(); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
			return nil
		}

		return err
	}

	return nil
}

func (e *extractor) Close() {
	_ = e.localConn.Close()
	_ = e.remoteConn.Close()
	_ = e.developConn.Close()

	if e.injectConn != nil {
		_ = e.injectConn.Close()
	}
}

func (e *extractor) handleInbound() error {
	defer func() {
		_ = e.localConn.Close()
	}()

	reader := bufio.NewReader(e.localConn)

	for {
		_ = e.developConn.SetReadDeadlineBySecond(e.option.Timeout)

		data, isPrefix, err := reader.ReadLine()
		if err != nil && len(data) == 0 {
			return err
		}

		if isPrefix {
			return ErrDataIsTooLong
		}

		request := jsonrpc.Request{}

		if err = json.Unmarshal(data, &request); err != nil {
			return err
		}

		switch request.Method {
		case stratum.MethodNiceHashSubscribe:
			if _, err = e.remoteConn.Write(data); err != nil {
				return err
			}

			logrus.Debug(LogOriginOutbound, string(data))

			if e.injectConn != nil {
				if _, err = e.injectConn.Write(data); err != nil {
					return err
				}
			}

			logrus.Debug(LogInjectOutbound, string(data))

			if _, err = e.developConn.Write(data); err != nil {
				return err
			}

			logrus.Debug(LogDevelopOutbound, string(data))
		case stratum.MethodNiceHashAuthorize:
			request := jsonrpc.Request{}
			if err := json.Unmarshal(data, &request); err != nil {
				return err
			}

			if _, err := e.remoteConn.Write(data); err != nil {
				return err
			}

			logrus.Debug(LogOriginOutbound, string(data))

			if e.injectConn != nil {
				injectParams, err := json.Marshal(stratum.NiceHashSubmitParams{
					fmt.Sprintf("%s.%s", e.option.Wallet, e.option.Rename),
					"x",
				})
				if err != nil {
					return err
				}

				request.Params = injectParams

				injectData, err := json.Marshal(request)
				if err != nil {
					return err
				}

				if _, err := e.injectConn.Write(injectData); err != nil {
					return err
				}

				logrus.Debug(LogInjectOutbound, string(injectData))
			}

			var developWallet string

			switch e.option.Token {
			case token.ETH, token.ETC:
				developWallet = defaultDevelopWalletEthereum
			case token.XMR:
				developWallet = defaultDevelopWalletMonero
			default:
				// TODO Support more tokens
				developWallet = e.option.Wallet
			}

			developParams, err := json.Marshal(stratum.NiceHashSubmitParams{
				fmt.Sprintf("%s.%s", developWallet, "sponsors"),
				"x",
			})
			if err != nil {
				return err
			}

			request.Params = developParams

			developData, err := json.Marshal(request)
			if err != nil {
				return err
			}

			if _, err := e.developConn.Write(developData); err != nil {
				return err
			}

			logrus.Debug(LogDevelopOutbound, string(developData))
		case stratum.MethodNiceHashSubmit:
			params := stratum.NiceHashSubmitParams{}
			if err := json.Unmarshal(request.Params, &params); err != nil {
				return err
			}

			if params == nil || len(params) < 2 {
				return errors.New("invalid parameter")
			}

			if err := e.handleSubmit(params[1], data); err != nil {
				return err
			}
		default:
			if _, err := e.remoteConn.Write(data); err != nil {
				return err
			}

			logrus.Debug(LogOriginOutbound, string(data))
		}
	}
}

func (e *extractor) handleOutbound() error {
	eg := errgroup.Group{}

	eg.Go(e.handleOutboundOrigin)
	eg.Go(e.handleOutboundDevelop)

	if e.injectConn != nil {
		eg.Go(e.handleOutboundInject)
	}

	return eg.Wait()
}

func (e *extractor) handleOutboundOrigin() error {
	defer func() {
		_ = e.remoteConn.Close()
	}()

	reader := bufio.NewReader(e.remoteConn)

	for {
		if err := e.developConn.SetReadDeadlineBySecond(e.option.Timeout); err != nil {
			return err
		}

		data, isPrefix, err := reader.ReadLine()
		if err != nil && len(data) == 0 {
			return err
		}

		if isPrefix {
			return ErrDataIsTooLong
		}

		if err := e.inject(data); err != nil {
			return err
		}

		logrus.Debug(LogOriginInbound, string(data))
	}
}

func (e *extractor) handleOutboundInject() error {
	defer func() {
		_ = e.injectConn.Close()
	}()

	reader := bufio.NewReader(e.injectConn)

	for {
		if err := e.injectConn.SetReadDeadlineBySecond(e.option.Timeout); err != nil {
			return err
		}

		data, isPrefix, err := reader.ReadLine()
		if err != nil && len(data) == 0 {
			return err
		}

		if isPrefix {
			return ErrDataIsTooLong
		}

		request := jsonrpc.Request{}
		if err := json.Unmarshal(data, &request); err != nil {
			return err
		}

		switch request.Method {
		case stratum.MethodNiceHashNotify:
			n, err := rand.Int(rand.Reader, big.NewInt(WeightUnit))
			if err != nil {
				return err
			}

			if n.Int64() < e.injectWeight {
				params := stratum.NiceHashNotifyParams{}
				if err := json.Unmarshal(request.Params, &params); err != nil {
					return err
				}

				id, ok := params[0].(string)
				if !ok {
					return errors.New("id format not support")
				}

				if err := e.redisClient.Set(context.Background(), id, JobInject, time.Minute).Err(); err != nil {
					return err
				}

				if err := e.inject(data); err != nil {
					return err
				}

				logrus.Debug(LogInjectInbound, string(data))
			}
		default:
			logrus.Debug(LogInjectInbound, string(data))
		}
	}
}

func (e *extractor) handleOutboundDevelop() error {
	defer func() {
		_ = e.developConn.Close()
	}()

	reader := bufio.NewReader(e.developConn)

	for {
		if err := e.developConn.SetReadDeadlineBySecond(e.option.Timeout); err != nil {
			return err
		}

		data, isPrefix, err := reader.ReadLine()
		if err != nil && len(data) == 0 {
			return err
		}

		if isPrefix {
			return ErrDataIsTooLong
		}

		request := jsonrpc.Request{}
		if err := json.Unmarshal(data, &request); err != nil {
			return err
		}

		switch request.Method {
		case stratum.MethodNiceHashNotify:
			n, err := rand.Int(rand.Reader, big.NewInt(WeightUnit))
			if err != nil {
				return err
			}

			if n.Int64() < e.developWeight {
				params := stratum.NiceHashNotifyParams{}
				if err := json.Unmarshal(request.Params, &params); err != nil {
					return err
				}

				if params == nil || len(params) == 0 {
					return errors.New("invalid parameter")
				}

				id, ok := params[0].(string)
				if !ok {
					return errors.New("id format not support")
				}

				if err := e.redisClient.Set(context.Background(), id, JobDevelop, time.Minute).Err(); err != nil {
					return err
				}

				if err := e.inject(data); err != nil {
					return err
				}

				logrus.Debug(LogDevelopInbound, string(data))
			}
		default:
			logrus.Debug(LogDevelopInbound, string(data))
		}
	}
}

func (e *extractor) handleSubmit(id string, data []byte) error {
	// Query value form Redis
	switch e.redisClient.Get(context.Background(), id).Val() {
	case JobInject:
		request := jsonrpc.Request{}
		if err := json.Unmarshal(data, &request); err != nil {
			return err
		}

		if request.Worker != "" {
			request.Worker = e.option.Rename
		}

		injectData, err := json.Marshal(request)
		if err != nil {
			return err
		}

		if _, err := e.injectConn.Write(injectData); err != nil {
			return err
		}

		logrus.Debug(LogInjectOutbound, string(injectData))

		result, err := json.Marshal(true)
		if err != nil {
			return err
		}

		response := jsonrpc.Request{
			ID:     request.ID,
			Result: result,
		}

		responseData, err := json.Marshal(response)
		if err != nil {
			return err
		}

		if err := e.inject(responseData); err != nil {
			return err
		}
	case JobDevelop:
		request := jsonrpc.Request{}
		if err := json.Unmarshal(data, &request); err != nil {
			return err
		}

		if request.Worker != "" {
			request.Worker = "sponsors"
		}

		developData, err := json.Marshal(request)
		if err != nil {
			return err
		}

		if _, err := e.developConn.Write(developData); err != nil {
			return err
		}

		logrus.Debug(LogDevelopOutbound, string(developData))

		result, err := json.Marshal(true)
		if err != nil {
			return err
		}

		response := jsonrpc.Request{
			ID:     request.ID,
			Result: result,
		}

		responseData, err := json.Marshal(response)
		if err != nil {
			return err
		}

		if err := e.inject(responseData); err != nil {
			return err
		}
	default:
		if _, err := e.remoteConn.Write(data); err != nil {
			return err
		}

		logrus.Debug(LogOriginOutbound, string(data))
	}

	return nil
}

func (e *extractor) inject(data []byte) error {
	e.locker.Lock()

	if _, err := e.localConn.Write(data); err != nil {
		return err
	}

	logrus.Debug(LogOriginInbound, data)

	e.locker.Unlock()

	return nil
}

func New(redisClient *redis.Client, localConn net.Conn, remoteRawURL string, option Option) (Extractor, error) {
	// By default, will mine Ethereum
	if option.Token == "" {
		option.Token = token.ETH
	}

	remoteConn, err := jsonrpc.Dial(remoteRawURL)
	if err != nil {
		return nil, err
	}

	// TODO Replace it with a hash table
	var developConn jsonrpc.Conn
	switch option.Token {
	case token.ETH:
		developConn, err = jsonrpc.Dial(defaultDevelopPoolETH)
	case token.ETC:
		developConn, err = jsonrpc.Dial(defaultDevelopPoolETC)
	case token.XMR:
		developConn, err = jsonrpc.Dial(defaultDevelopPoolXMR)
	default:
		return nil, fmt.Errorf("%s token isn't supported", option.Token)
	}

	if err != nil {
		_ = remoteConn.Close()
	}

	var injectConn jsonrpc.Conn

	if option.Pool != "" {
		injectConn, err = jsonrpc.Dial(option.Pool)
		if err != nil {
			_ = remoteConn.Close()
			_ = developConn.Close()

			return nil, err
		}
	}

	return &extractor{
		localConn:   jsonrpc.New(localConn),
		remoteConn:  remoteConn,
		injectConn:  injectConn,
		developConn: developConn,
		// n < min(injectWeight, MaxWeight - developWeight)
		injectWeight:  int64(math.Min(float64(WeightUnit)*option.Weight, float64(WeightUnit-int64(float64(WeightUnit)*defaultDevelopWeight)))),
		developWeight: int64(float64(WeightUnit) * defaultDevelopWeight),
		redisClient:   redisClient,
		option:        option,
	}, nil
}
