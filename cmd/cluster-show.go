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
  "strings"
  
  "github.com/spf13/cobra"
  
  "github.com/alces-software/flight-attendant/attendant"
)

// launchCmd represents the launch command
var clusterShowCmd = &cobra.Command{
  Use:   "show <name>",
  Short: "Show details of a running Flight Compute cluster",
  Long: `Show details of a running Flight Compute cluster.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    if len(args) < 1 {
      cmd.Help()
      return nil
    }

    var domain *attendant.Domain
    var err error

    if err := attendant.PreflightCheck(); err != nil { return err }
    solo, _ := cmd.Flags().GetBool("solo")
    if solo {
      var status *attendant.DomainStatus
      var details string
      attendant.SpinWithSuffix(func() { status, err = attendant.SoloStatus() }, attendant.Config().AwsRegion + " (Solo)")
      found := false
      for _, cluster := range status.Clusters {
        if cluster.Name == args[0] {
          attendant.SpinWithSuffix(func() { details = cluster.GetDetails() }, attendant.Config().AwsRegion + ": " + args[0])
          fmt.Println(cluster.Name)
          fmt.Println(strings.Repeat("-", len(cluster.Name)))
          fmt.Println(details)
          found = true
          break
        }
      }
      if !found {
        return fmt.Errorf("Solo cluster not found: %s (%s)", args[0], attendant.Config().AwsRegion)
      } else {
        return nil
      }
    } else {
      domain, err = findDomain("clusterShow", false)
      if err != nil { return err }

      cluster := attendant.NewCluster(args[0], domain, nil)
      var exists bool
      attendant.SpinWithSuffix(func() { exists = cluster.Exists() }, attendant.Config().AwsRegion + ": " + cluster.Domain.Name + "/" + cluster.Name)
      if exists {
        fmt.Println(cluster.Name)
        fmt.Println(strings.Repeat("-", len(cluster.Name)))

        var details string
        attendant.SpinWithSuffix(func() { details = cluster.GetDetails() }, attendant.Config().AwsRegion + ": " + cluster.Domain.Name + "/" + cluster.Name)
        fmt.Println(details)
        return nil
      } else {
        return fmt.Errorf("Cluster not found: %s/%s (%s)", cluster.Domain.Name, cluster.Name, attendant.Config().AwsRegion)
      }
    }
  },
}

func init() {
  clusterCmd.AddCommand(clusterShowCmd)
  addDomainFlag(clusterShowCmd, "clusterShow")
  clusterShowCmd.Flags().BoolP("solo", "s", false, "Show Flight Compute Solo cluster")
}

