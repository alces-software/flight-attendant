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
  Use:   "fly",
  Short: "Helper utility for lauching Alces Flight Enterprise clusters",
  Long: `Alces Flight Attendant is a command-line helper utility that makes it
quick and easy to launch Alces Flight enterprise infrastructure
appliances, launch and manage clusters and get status information
on your Alces Flight Compute architecture.`,
// Uncomment the following line if your bare application
  // has an action associated with it:
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    if v, _ := cmd.Flags().GetBool("version"); v {
      fmt.Printf("Flight Attendant v%s %s (Flight Compute %s)\n", attendant.Version, attendant.ReleaseDate, attendant.FlightRelease)
    } else if v, _ := cmd.Flags().GetBool("show-config-example"); v {
      cfg, err := attendant.RenderConfig()
      if err != nil { return err }
      fmt.Println(string(cfg))
    } else if v, _ := cmd.Flags().GetBool("show-config-values"); v {
      cfg, err := attendant.RenderConfigValues()
      if err != nil { return err }
      fmt.Println(cfg)
    } else if v, _ := cmd.Flags().GetString("create-parameter-directory"); v != "" {
      err := attendant.CreateParameterDirectory(v)
      if err != nil { return err }
    } else {
      cmd.Help()
    }
    return nil
  },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
  if err := RootCmd.Execute(); err != nil {
    // fmt.Println(err.Error())
    os.Exit(-255)
  }
}

func init() {
  cobra.OnInitialize(initConfig)
  attendant.Init()
  RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fly.yml)")
  defaultRegion := os.Getenv("AWS_REGION")
  if defaultRegion == "" { defaultRegion = "us-east-1" }
  RootCmd.Flags().Bool("version", false, "Show version information")
  RootCmd.PersistentFlags().String("region", defaultRegion, "AWS region")
  RootCmd.PersistentFlags().String("access-key", "", "AWS access key ID")
  RootCmd.PersistentFlags().String("secret-key", "", "AWS secret access key")
  RootCmd.PersistentFlags().String("parameter-directory", "", "Directory containing component parameter files")
  RootCmd.Flags().Bool("show-config-example", false, "Display an example configuration file")
  RootCmd.Flags().Bool("show-config-values", false, "Display valid configuration values")
  RootCmd.Flags().String("create-parameter-directory", "", "Write default parameter files to a directory")

  viper.BindPFlag("region", RootCmd.PersistentFlags().Lookup("region"))
  viper.BindPFlag("access-key", RootCmd.PersistentFlags().Lookup("access-key"))
  viper.BindPFlag("secret-key", RootCmd.PersistentFlags().Lookup("secret-key"))
  viper.BindPFlag("parameter-directory", RootCmd.PersistentFlags().Lookup("parameter-directory"))

  for key, val := range attendant.ConfigDefaults {
    viper.SetDefault(key, val)
  }
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
  viper.SetConfigName(".fly") // name of config file (without extension)
  viper.AddConfigPath("$HOME")  // adding home directory as first search path
  viper.SetEnvPrefix("FLY")
  replacer := strings.NewReplacer("-", "_")
  viper.SetEnvKeyReplacer(replacer)
  viper.AutomaticEnv()          // read in environment variables that match

  if cfgFile != "" { // enable ability to specify config file via flag
    viper.SetConfigFile(cfgFile)
  } else if os.Getenv("FLY_CONFIG") != "" {
    viper.SetConfigFile(os.Getenv("FLY_CONFIG"))
  }

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

  cfg.ParameterDirectory = viper.GetString("parameter-directory")
}

func addDomainFlag(command *cobra.Command, cmdName string) {
  command.Flags().StringP("domain", "d", "", "Domain for cluster or infrastructure appliance")
  viper.BindPFlag("domain:" + cmdName, command.Flags().Lookup("domain"))
}

func findDomain(cmdName string, defaultOk bool) (*attendant.Domain, error) {
  var domain *attendant.Domain
  var err error
  name := viper.GetString("domain:" + cmdName)
  if name == "" { name = viper.GetString("domain") }
  if name == "" {
    if defaultOk {
      domain, err = attendant.DefaultDomain()
    } else {
      return nil, fmt.Errorf("This operation requires you to specify a domain")
    }
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

func addTemplateSetFlag(command *cobra.Command, cmdName string) {
  command.Flags().String("template-set", "", "Select a predefined template set")
  viper.BindPFlag("template-set:" + cmdName, command.Flags().Lookup("template-set"))
}

func addTemplateRootFlag(command *cobra.Command, cmdName string) {
  command.Flags().String("template-root", "", "Specify an explicit template root URL")
  viper.BindPFlag("template-root:" + cmdName, command.Flags().Lookup("template-root"))
}

func setupTemplateSource(cmdName string) error {
  templateRoot := viper.GetString("template-root:" + cmdName)
  templateSet := viper.GetString("template-set:" + cmdName)
  if templateRoot != "" && templateSet != "" {
    return fmt.Errorf("Template set cannot be used in conjunction with template root\n")
  } else {
    if templateRoot == "" { templateRoot = viper.GetString("template-root") }
    if templateSet == "" { templateSet = viper.GetString("template-set") }
    if templateRoot != "" {
      attendant.Config().TemplateRoot = templateRoot
      attendant.Config().TemplateSet = ""
    } else if templateSet != "" {
      attendant.Config().TemplateSet = templateSet
    }
  }
  return nil
}

func getRegions(cmd *cobra.Command) []string {
  var regions []string
  regionSelector, _ := cmd.Flags().GetString("regions")
  if regionSelector == "all" {
    regions = attendant.AwsRegions
  } else if regionSelector == "" {
    regions = []string { attendant.Config().AwsRegion }
  } else {
    regions = strings.Split(regionSelector,",")
  }
  return regions
}
