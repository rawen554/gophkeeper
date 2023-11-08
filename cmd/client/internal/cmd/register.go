package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/rawen554/goph-keeper/cmd/client/internal/logic"
	"github.com/rawen554/goph-keeper/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(registerCmd)
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register to gophkeeper",
	Run: func(cmd *cobra.Command, args []string) {
		Register(context.Background())
	},
}

func Register(ctx context.Context) {
	fmt.Println("Login:")
	var login string
	fmt.Scanln(&login)

	fmt.Println("Password:")
	var password string
	fmt.Scanln(&password)

	creds, err := logic.Register(login, password)
	if err != nil {
		var target *net.OpError
		if errors.As(err, &target) {
			if err := utils.CreateUsersDir(login); err != nil {
				fmt.Println("err: %w", err)
			}
			fmt.Printf("created local dir for user: %s\n", login)
		}

		return
	}

	viper.Set("login", login)
	viper.Set("token", creds.Token)
	viper.Set("expires_at", time.Now().Add(time.Duration(creds.ExpiresIn)*time.Second))

	if err := utils.CreateUsersDir(login); err != nil {
		fmt.Println("err: %w", err)
	}

	if err := viper.WriteConfigAs("./gophkeeper.json"); err != nil {
		fmt.Println("err saving config: %w", err)
	}
}
