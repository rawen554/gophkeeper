package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile string
	apiURL  string

	rootCmd = &cobra.Command{
		Use:   "gclient",
		Short: "A generator for Cobra based Applications",
		Long: `Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api", "https://localhost:8080", "API URL")
	rootCmd.PersistentFlags().StringP("login", "l", "", "author name for copyright attribution")
	rootCmd.PersistentFlags().StringP("token", "t", "", "author name for copyright attribution")
	viper.BindPFlag("api", rootCmd.PersistentFlags().Lookup("api"))
	viper.BindPFlag("login", rootCmd.PersistentFlags().Lookup("login"))
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("expires_at", rootCmd.PersistentFlags().Lookup("expires_at"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("json")
		viper.SetConfigName("gophkeeper")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
