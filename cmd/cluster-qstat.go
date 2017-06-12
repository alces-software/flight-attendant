// Copyright © 2017 Alces Software Ltd <support@alces-software.com>
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
  "strings"
  "time"

  "github.com/spf13/cobra"
  
  "github.com/alces-software/flight-attendant/attendant"
)

// launchCmd represents the launch command
var clusterQstatCmd = &cobra.Command{
  Use:   "qstat <cluster> [<name>]",
  Short: "Query for information about a compute queue on a running Flight Compute cluster",
  Long: `Query for information about a compute queue on a running Flight Compute cluster.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    if len(args) < 1 {
      cmd.Help()
      return nil
    }

    var domain *attendant.Domain
    var err error

    if err := attendant.PreflightCheck(); err != nil { return err }
    domain, err = findDomain("clusterQstat", false)
    if err != nil { return err }

    cluster := attendant.NewCluster(args[0], domain, nil)
    var exists bool
    attendant.SpinWithSuffix(func() {
      exists = cluster.Exists()
      if exists { err = cluster.LoadComputeGroups() }
    }, attendant.Config().AwsRegion + ": " + cluster.Domain.Name + "/" + cluster.Name)
    if exists {
      if err != nil { return err }
      if len(cluster.ComputeGroups) == 0 {
        return fmt.Errorf("No compute queues running on cluster: %s/%s (%s)", cluster.Domain.Name, cluster.Name, attendant.Config().AwsRegion)
      }
      if len(args) > 1 {
        var foundGroup *attendant.ComputeGroup
        for  _, group := range cluster.ComputeGroups {
          if group.Name == args[1] {
            foundGroup = group
            break
          }
        }
        if foundGroup != nil {
          fmt.Println("== " + cluster.Domain.Name + "/" + cluster.Name + " (" + attendant.Config().AwsRegion + ") ==")
          fmt.Println()
          showGroupDetails(foundGroup)
        } else {
          return fmt.Errorf("No compute queue named '%s' running on cluster: %s/%s (%s)", args[1], cluster.Domain.Name, cluster.Name, attendant.Config().AwsRegion)
        }
      } else {
        fmt.Println("== " + cluster.Domain.Name + "/" + cluster.Name + " (" + attendant.Config().AwsRegion + ") ==")
        for  _, group := range cluster.ComputeGroups {
          fmt.Println()
          showGroupDetails(group)
        }
      }
      return nil
    } else {
      return fmt.Errorf("Cluster not found: %s/%s (%s)", cluster.Domain.Name, cluster.Name, attendant.Config().AwsRegion)
    }
  },
}

func init() {
  clusterCmd.AddCommand(clusterQstatCmd)
  addDomainFlag(clusterQstatCmd, "clusterQstat")
}

func showGroupDetails(group *attendant.ComputeGroup) {
  fmt.Println("    " + group.Name)
  fmt.Println("    " + strings.Repeat("-", len(group.Name)))
  fmt.Println("        Type: " + group.InstanceType)
  fmt.Println("     Pricing: " + group.Pricing)
  fmt.Printf("    Capacity: %d-%d\n", group.MinSize(), group.MaxSize())
  fmt.Printf("     Running: %d\n", group.Running())
  fmt.Printf("     Pending: %d\n", group.DesiredCapacity() - group.Running())
  if group.ExpiryTime > 0 {
    fmt.Printf("      Expiry: %s\n", time.Unix(group.ExpiryTime, 0).Format(time.RFC3339))
  }
  // fmt.Printf("    Resource: %s\n", group.ResourceName)
}
