package cmd

import (
	"context"
	"log"

	"github.com/rawen554/goph-keeper/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(LogoutCmd)
}

var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from gophkeeper",
	Run: func(cmd *cobra.Command, args []string) {
		logger, err := logger.NewLogger()
		if err != nil {
			log.Fatal(err)
		}

		Logout(context.Background(), logger)
	},
}

func Logout(ctx context.Context, logger *zap.SugaredLogger) {
	login := viper.GetString("login")
	if login == "" {
		logger.Errorln("not logged in")
		return
	}

	viper.Set("login", "")
	viper.Set("token", "")
	viper.Set("expires_at", "")

	if err := viper.WriteConfigAs("./gophkeeper.json"); err != nil {
		logger.Errorf("err saving config: %w", err)
	}

	logger.Infof("cleared session: %s\n", login)
}
