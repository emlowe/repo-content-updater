package cmd

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var fs embed.FS
var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "repo-content-updater",
	Short: "Keeps known files in a repo up to date",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(_fs embed.FS) {
	fs = _fs
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	var (
		githubToken string
		signCommits bool
		push        bool
	)

	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.repo-content-updater.yaml)")

	rootCmd.PersistentFlags().StringVar(&githubToken, "github-token", "", "The token to use to auth to GitHub API and Push to Repos")
	rootCmd.PersistentFlags().BoolVar(&signCommits, "sign-commits", true, "Whether or not to sign commits")
	rootCmd.PersistentFlags().BoolVar(&push, "push", true, "Whether or not to push and create the pull request")

	cobra.CheckErr(viper.BindPFlag("github-token", rootCmd.PersistentFlags().Lookup("github-token")))
	cobra.CheckErr(viper.BindPFlag("sign-commits", rootCmd.PersistentFlags().Lookup("sign-commits")))
	cobra.CheckErr(viper.BindPFlag("push", rootCmd.PersistentFlags().Lookup("push")))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".repo-content-updater" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".repo-content-updater")
	}

	viper.SetEnvPrefix("REPO_CONTENT_UPDATER")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
