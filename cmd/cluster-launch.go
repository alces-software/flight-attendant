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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/alces-software/flight-attendant/attendant"
)

// launchCmd represents the launch command
var clusterLaunchCmd = &cobra.Command{
	Use:   "launch <name>",
	Short: "Launch a Flight Compute cluster",
	Long: `Launch a Flight Compute cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}

    masterInstanceType := viper.GetString("master-instance-type")
    if masterInstanceType != "" {
      if ! attendant.IsValidMasterInstanceType(masterInstanceType) {
        fmt.Printf("Invalid master instance type '%s'. Try one of: %s\n", masterInstanceType, attendant.MasterInstanceTypes)
        return
      }
    }

    computeInstanceType := viper.GetString("compute-instance-type")
    if computeInstanceType != "" {
      if ! attendant.IsValidComputeInstanceType(computeInstanceType) {
        fmt.Printf("Invalid compute instance type '%s'. Try one of: %s\n", computeInstanceType, attendant.ComputeInstanceTypes)
        return
      }
    }

    if err := setupKeyPair("clusterLaunch"); err != nil {
      fmt.Println(err.Error())
      return
    }

    domain, err := findDomain("clusterLaunch")
    if err != nil {
      fmt.Println(err.Error())
      return
    }

    if err = domain.AssertReady(); err != nil {
      fmt.Println("Domain is not ready: " + domain.Name)
      return
    }

    fmt.Printf("Launching cluster '%s' in domain '%s' (%s)...\n\n", args[0], domain.Name, attendant.Config().AwsRegion)
    cluster, err := launchCluster(domain, args[0])
		if err != nil {
			fmt.Println(err.Error())
			return
		}
    fmt.Println("\nCluster launched.\n")
    fmt.Println("== Access details ==")
    fmt.Println(cluster.GetAccessDetails() + "\n")
    fmt.Println("\nAccess via:\n\n\tssh " + cluster.Master.Username() + "@" + cluster.Master.AccessIP())
	},
}

func init() {
	clusterCmd.AddCommand(clusterLaunchCmd)

  clusterLaunchCmd.Flags().StringP("compute-instance-type", "c", attendant.ComputeInstanceTypes[0], "Compute instance type")
  viper.BindPFlag("compute-instance-type", clusterLaunchCmd.Flags().Lookup("compute-instance-type"))

  clusterLaunchCmd.Flags().StringP("master-instance-type", "m", attendant.MasterInstanceTypes[0], "Master instance type")
  viper.BindPFlag("master-instance-type", clusterLaunchCmd.Flags().Lookup("master-instance-type"))

  addKeyPairFlag(clusterLaunchCmd, "clusterLaunch")
  addDomainFlag(clusterLaunchCmd, "clusterLaunch")
}

func launchCluster(domain *attendant.Domain, name string) (*attendant.Cluster, error) {
  handler, err := attendant.CreateCreateHandler(attendant.ClusterResourceCount)
  if err != nil { return nil, err }
  cluster := attendant.NewCluster(name, domain, handler)
  attendant.Spin(func() { err = cluster.Create() })
  cluster.MessageHandler = nil
  return cluster, err
}
