package server

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tier2pool/tier2pool/internal/command"
	"github.com/tier2pool/tier2pool/internal/extractor"
)

var _ command.Interface = &Server{}

type Server struct {
	command     *cobra.Command
	config      Config
	redisClient *redis.Client
	listener    net.Listener
}

func (s *Server) Initialize(cmd *cobra.Command) error {
	s.command = cmd

	// Enable debug mode
	if debug, err := s.command.Flags().GetBool("debug"); err == nil && debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := s.initializeRedis(); err != nil {
		return err
	}

	logrus.Info("initialization completed")

	return nil
}

func (s *Server) initializeRedis() error {
	ip, err := net.ResolveTCPAddr("tcp", s.config.Redis.Address)
	if err != nil {
		return err
	}

	// Redis is exposed to the public network without protection
	if s.config.Redis.Password == "" && !(ip.IP.IsPrivate() || ip.IP.IsLoopback()) {
		logrus.Warn("redis has no password and is exposed to the public network")
	}

	s.redisClient = redis.NewClient(&redis.Options{
		Addr:     s.config.Redis.Address,
		Password: s.config.Redis.Password,
	})

	// Check if the connection is successful
	if err := s.redisClient.Ping(context.Background()).Err(); err != nil {
		return err
	}

	logrus.Info("connected to redis")

	return nil
}

func (s *Server) Run(cmd *cobra.Command, _ []string) (err error) {
	if err = s.Initialize(cmd); err != nil {
		return err
	}

	// Nginx or other gateways are flexible options
	if s.config.Server.TLS == nil {
		if s.listener, err = net.Listen("tcp", s.config.Server.Address); err != nil {
			return err
		}
	} else {
		certificate, err := tls.LoadX509KeyPair(
			s.config.Server.TLS.Certificate,
			s.config.Server.TLS.PrivateKey,
		)
		if err != nil {
			return err
		}

		s.listener, err = tls.Listen("tcp", s.config.Server.Address, &tls.Config{
			Certificates: []tls.Certificate{
				certificate,
			},
		})
		if err != nil {
			return err
		}
	}

	var conn net.Conn
	for {
		conn, err = s.listener.Accept()
		if err != nil {
			logrus.Error(err)

			continue
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(localConn net.Conn) {
	logrus.Infof("new connection from %s", localConn.RemoteAddr())

	defer logrus.Infof("%s is disconnected", localConn.RemoteAddr())

	extractorConfig := extractor.Option{
		Token:   s.config.Pool.Token,
		Timeout: s.config.Server.Timeout,
	}

	// Users may choose to use only for forwarding
	if s.config.Pool.Inject != nil {
		extractorConfig.Pool = s.config.Pool.Inject.Pool
		extractorConfig.Wallet = s.config.Pool.Inject.Wallet
		extractorConfig.Weight = s.config.Pool.Inject.Weight
		extractorConfig.Rename = s.config.Pool.Inject.Rename
	}

	conn, err := extractor.New(s.redisClient, localConn, s.config.Pool.Default, extractorConfig)
	if err != nil {
		logrus.Error(err)

		return
	}

	if err := conn.Inject(); err != nil {
		logrus.Error(err)

		return
	}
}

func NewCommand() *cobra.Command {
	srv := Server{}

	cmd := cobra.Command{
		Use:  "server",
		RunE: srv.Run,
	}

	cmd.Flags().StringP("config", "c", "server", "config file name")

	viper.SetConfigName(cmd.Flag("config").Value.String())

	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatal(err)
	}

	if err := viper.Unmarshal(&srv.config); err != nil {
		logrus.Fatal(err)
	}

	return &cmd
}
