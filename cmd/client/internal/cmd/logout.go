package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(LogoutCmd)
}

var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from gophkeeper",
	Run: func(cmd *cobra.Command, args []string) {
		Logout(context.Background())
	},
}

func Logout(ctx context.Context) {
	login := viper.GetString("login")
	if login == "" {
		fmt.Println("not logged in")
		return
	}

	viper.Set("login", "")
	viper.Set("token", "")
	viper.Set("expires_at", "")

	if err := viper.WriteConfigAs("./gophkeeper.json"); err != nil {
		fmt.Println("err saving config: %w", err)
	}

	fmt.Printf("cleared session: %s\n", login)
}
