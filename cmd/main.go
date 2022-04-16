package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tier2pool/tier2pool/cmd/server"
	"github.com/tier2pool/tier2pool/internal/flag"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	viper.AutomaticEnv()
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/tier2pool")
	viper.AddConfigPath("$HOME/.config/tier2pool")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
}

func main() {
	cmd := cobra.Command{
		Use:     "tier2pool",
		Version: flag.Version,
	}

	// Human sorrows and joys are not interlinked, I just think they are noisy
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	cmd.PersistentFlags().BoolP("debug", "d", false, "debug mode")

	cmd.AddCommand(server.NewCommand())

	if err := cmd.Execute(); err != nil {
		logrus.Fatalln(err)
	}
}
