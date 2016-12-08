// Copyright © 2016 Alces Software Ltd <support@alces-software.com>
// This file is part of Flight Attendant.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This software is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public
// License along with this software.  If not, see
// <http://www.gnu.org/licenses/>.
//
// This package is available under a dual licensing model whereby use of
// the package in projects that are licensed so as to be compatible with
// AGPL Version 3 may use the package under the terms of that
// license. However, if AGPL Version 3.0 terms are incompatible with your
// planned use of this package, alternative license terms are available
// from Alces Software Ltd - please direct inquiries about licensing to
// licensing@alces-software.com.
//
// For more information, please visit <http://www.alces-software.com/>.
//

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/alces-software/flight-attendant/attendant"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "flight-attendant",
	Short: "Helper utility for lauching Alces Flight Enterprise clusters",
	Long: `Alces Flight Attendant is a command-line helper utility that makes it
quick and easy to launch Alces Flight enterprise infrastructure
appliances, launch and manage clusters and get status information
on your Alces Flight Compute architecture.`,
// Uncomment the following line if your bare application
// has an action associated with it:
//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
  attendant.Init()
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.flight.yml)")
  defaultRegion := os.Getenv("AWS_REGION")
  if defaultRegion == "" { defaultRegion = "us-east-1" }
  RootCmd.PersistentFlags().String("region", defaultRegion, "AWS region")
  RootCmd.PersistentFlags().String("access-key", "", "AWS access key ID")
  RootCmd.PersistentFlags().String("secret-key", "", "AWS secret access key")
  RootCmd.PersistentFlags().String("template-set", "default", "Template set")
  viper.BindPFlag("region", RootCmd.PersistentFlags().Lookup("region"))
  viper.BindPFlag("access-key", RootCmd.PersistentFlags().Lookup("access-key"))
  viper.BindPFlag("secret-key", RootCmd.PersistentFlags().Lookup("secret-key"))
  viper.BindPFlag("template-set", RootCmd.PersistentFlags().Lookup("template-set"))

  viper.SetDefault("admin-user-name", "alces")
  viper.SetDefault("access-network", "0.0.0.0/0")
  viper.SetDefault("scheduler-type", "gridscheduler")
  viper.SetDefault("profile-bucket", "")

  viper.SetDefault("directory-profiles", "")
  viper.SetDefault("directory-features", "")
  viper.SetDefault("directory-instance-type", "small-t2.large")

  viper.SetDefault("access-manager-profiles", "")
  viper.SetDefault("access-manager-features", "")
  viper.SetDefault("access-manager-instance-type", "small-t2.large")

  viper.SetDefault("storage-manager-profiles", "")
  viper.SetDefault("storage-manager-features", "")
  viper.SetDefault("storage-manager-instance-type", "small-t2.large")

  viper.SetDefault("master-profiles", "")
  viper.SetDefault("master-features", "")
  viper.SetDefault("master-instance-override", "")
  viper.SetDefault("preload-software", "")
  viper.SetDefault("master-volume-layout", "")
  viper.SetDefault("master-volume-encryption-policy", "unencrypted")
  viper.SetDefault("master-system-volume-size", "500")
  viper.SetDefault("master-system-volume-type", "")
  viper.SetDefault("master-home-volume-size", "")
  viper.SetDefault("master-home-volume-type", "")
  viper.SetDefault("master-apps-volume-size", "")
  viper.SetDefault("master-apps-volume-type", "")

  viper.SetDefault("compute-profiles", "")
  viper.SetDefault("compute-features", "")
  viper.SetDefault("compute-instance-override", "")
  viper.SetDefault("compute-spot-price", "")
  viper.SetDefault("compute-autoscaling-policy", "enabled")
  viper.SetDefault("compute-initial-nodes", "1")
  viper.SetDefault("compute-system-volume-type", "")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".flight") // name of config file (without extension)
	viper.AddConfigPath("$HOME")  // adding home directory as first search path
  viper.SetEnvPrefix("FLIGHT")
  replacer := strings.NewReplacer("-", "_")
  viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()          // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		//fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

  cfg := attendant.Config()
  cfg.AwsRegion = viper.GetString("region")
  if cfg.AwsRegion == "" {
    cfg.AwsRegion = os.Getenv("AWS_REGION")
  }

  cfg.AwsAccessKey = viper.GetString("access-key")
  if cfg.AwsAccessKey == "" {
    cfg.AwsAccessKey = os.Getenv("AWS_ACCESS_KEY_ID")
  }

  cfg.AwsSecretKey = viper.GetString("secret-key")
  if cfg.AwsSecretKey == "" {
     cfg.AwsSecretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
  }

  if attendant.TemplateSets[viper.GetString("template-set")] != "" {
    cfg.TemplateSet = viper.GetString("template-set")
  }
}

func addDomainFlag(command *cobra.Command, cmdName string) {
  command.Flags().StringP("domain", "d", "", "Domain for cluster or infrastructure appliance")
  viper.BindPFlag("domain:" + cmdName, command.Flags().Lookup("domain"))
}

func findDomain(cmdName string) (*attendant.Domain, error) {
  var domain *attendant.Domain
  var err error
  name := viper.GetString("domain:" + cmdName)
  if name == "" { name = viper.GetString("domain") }
  if name == "" {
    domain, err = attendant.DefaultDomain()
  } else {
    domain = attendant.NewDomain(name, nil)
    err = domain.AssertExists()
  }
  if err != nil {
    return nil, err
  }
  return domain, nil
}

func addKeyPairFlag(command *cobra.Command, cmdName string) {
  command.Flags().StringP("key-pair", "k", "", "EC2 key pair name (default: \"flight-admin\")")
  viper.BindPFlag("key-pair:" + cmdName, command.Flags().Lookup("key-pair"))
}

func setupKeyPair(cmdName string) error {
  keyPairName := viper.GetString("key-pair:" + cmdName)
  if keyPairName == "" { keyPairName = viper.GetString("key-pair") }
  if keyPairName != "" {
    attendant.Config().AccessKeyName = keyPairName
  }
  if ! attendant.Config().IsValidKeyPair() {
    return fmt.Errorf("Invalid key pair name '%s'.\n", attendant.Config().AccessKeyName)
  }
  return nil
}
