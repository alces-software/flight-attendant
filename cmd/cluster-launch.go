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
  "time"
  "github.com/spf13/cobra"
  "github.com/spf13/viper"

  "github.com/alces-software/flight-attendant/attendant"
)

// launchCmd represents the launch command
var clusterLaunchCmd = &cobra.Command{
  Use:   "launch <name>",
  Short: "Launch a Flight Compute cluster",
  Long: `Launch a Flight Compute cluster.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    if len(args) == 0 {
      cmd.Help()
      return nil
    }

    masterInstanceType := viper.GetString("master-instance-type")
    if masterInstanceType != "" {
      if ! attendant.IsValidMasterInstanceType(masterInstanceType) {
        return fmt.Errorf("Invalid master instance type '%s'. Try one of: %s\n", masterInstanceType, attendant.MasterInstanceTypes)
      }
    }

    withQ, _ := cmd.Flags().GetBool("with-queue")
    if withQ {
      queueInstanceType := viper.GetString("queue-instance-type")
      if queueInstanceType != "" {
        if ! attendant.IsValidComputeInstanceType(queueInstanceType) {
          return fmt.Errorf("Invalid compute instance type '%s'. Try one of: %s\n", queueInstanceType, attendant.ComputeInstanceTypes)
        }
      }
    }

    var expiryTime int64

    runtime, _ := cmd.Flags().GetInt("runtime")
    if runtime > 0 {
      duration, err := time.ParseDuration(fmt.Sprintf("%dm",runtime))
      if err != nil { return err }
      expiryTime = time.Now().Add(duration).Unix()
    }
    
    if err := attendant.PreflightCheck(); err != nil { return err }
    if err := setupTemplateSource("clusterLaunch"); err != nil { return err }
    if err := setupKeyPair("clusterLaunch"); err != nil { return err }

    var cluster *attendant.Cluster
    var domain *attendant.Domain
    var err error
    solo, _ := cmd.Flags().GetBool("solo")
    if solo {
      fmt.Printf("Launching Flight Compute Solo cluster '%s' (%s)...\n\n", args[0], attendant.Config().AwsRegion)
      domain = nil
    } else {
      domain, err = findDomain("clusterLaunch", true)
      if err != nil { return err }

      if err = domain.AssertReady(); err != nil {
        return fmt.Errorf("Domain is not ready: " + domain.Name)
      }

      fmt.Printf("Launching cluster '%s' in domain '%s' (%s)...\n\n", args[0], domain.Name, attendant.Config().AwsRegion)
    }
    cluster, err = launchCluster(domain, args[0], withQ, expiryTime)
    if err != nil { return err }

    fmt.Println("\nCluster launched.\n")
    fmt.Println("== Cluster details ==")
    fmt.Println(cluster.GetDetails() + "\n")
    ip := cluster.Master.AccessIP()
    if ip == "" {
      ip = cluster.Master.PrivateIP()
    }
    fmt.Println("\nAccess via:\n\n\tssh " + cluster.Master.Username() + "@" + ip)
    return nil
  },
}

func init() {
  clusterCmd.AddCommand(clusterLaunchCmd)
  clusterLaunchCmd.Flags().BoolP("solo", "s", false, "Launch a Flight Compute Solo cluster")
  clusterLaunchCmd.Flags().IntP("runtime", "r", 0, "Maximum runtime for cluster (hours)")

  clusterLaunchCmd.Flags().BoolP("with-queue", "q", false, "Launch with a compute queue")
  viper.BindPFlag("launch-with-default-queue", clusterLaunchCmd.Flags().Lookup("with-queue"))
  clusterLaunchCmd.Flags().StringP("queue-instance-type", "t", attendant.ComputeInstanceTypes[0], "Compute queue instance type")
  viper.BindPFlag("default-queue-instance-type", clusterLaunchCmd.Flags().Lookup("queue-instance-type"))

  clusterLaunchCmd.Flags().StringP("master-instance-type", "m", attendant.MasterInstanceTypes[0], "Master instance type")
  viper.BindPFlag("master-instance-type", clusterLaunchCmd.Flags().Lookup("master-instance-type"))

  addKeyPairFlag(clusterLaunchCmd, "clusterLaunch")
  addDomainFlag(clusterLaunchCmd, "clusterLaunch")
  addTemplateSetFlag(clusterLaunchCmd, "clusterLaunch")
  addTemplateRootFlag(clusterLaunchCmd, "clusterLaunch")
}

func launchCluster(domain *attendant.Domain, name string, withQ bool, expiryTime int64) (*attendant.Cluster, error) {
  var count int
  if domain == nil {
    count = attendant.SoloClusterResourceCount
  } else {
    count = attendant.ClusterResourceCount
    if withQ {
      count += attendant.ComputeGroupResourceCount
    }
  }
  handler, err := attendant.CreateCreateHandler(count)
  if err != nil { return nil, err }
  cluster := attendant.NewCluster(name, domain, handler)
  cluster.ExpiryTime = expiryTime
  attendant.Spin(func() { err = cluster.Create(withQ) })
  cluster.MessageHandler = nil
  return cluster, err
}
