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

// clusterCmd represents the cluster command
var cleanupCmd = &cobra.Command{
  Use:   "cleanup",
  Short: "Clean up Alces Flight resources",
  Long: `Clean up Alces Flight resources.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    var domains []attendant.Domain
    var err error
    regions := getRegions(cmd)
    for _, region := range regions {
      attendant.Config().AwsRegion = region
      attendant.SpinWithSuffix(func() {
        domains, err = attendant.AllDomains()
      }, region)
      if err != nil { return err }
      var stacks = []string{}
      for _, domain := range domains {
        var status *attendant.DomainStatus
        var networkIndices = []int{}
        attendant.SpinWithSuffix(func() { status, err = domain.Status() }, region + ": " + domain.Name)
        if err != nil { return err }
        stacks = append(stacks, "flight-" + domain.Name)
        // list all topics, subscriptions, queues and remove any that aren't accounted for
        for _, cluster := range status.Clusters {
          stacks = append(stacks, "flight-" + domain.Name + "-cluster-" + cluster.Name)
          networkIndices = append(networkIndices, cluster.Network.Index)
        }
        for _, appliance := range status.Appliances {
          stacks = append(stacks, "flight-" + domain.Name + "-" + appliance.Name)
        }
        entity, err := domain.LoadEntity()
        if err != nil { return err }
        for _, booking := range entity.NetBookings {
          for _, a := range networkIndices {
            if a == booking {
              break
            }
            fmt.Printf("🗑  Purge stale network booking: %s/%d\n", domain.Name, booking)
            if dryrun, _ := cmd.Flags().GetBool("dry-run"); !dryrun {
              domain.ReleaseNetwork(booking)
            }
          }
        }
      }
      var soloStatus *attendant.DomainStatus
      attendant.SpinWithSuffix(func() { soloStatus, err = attendant.SoloStatus() }, region + " (Solo)")
      if err != nil { return err }
      for _, cluster := range soloStatus.Clusters {
        stacks = append(stacks, "flight-cluster-" + cluster.Name)
      }

      if ( len(stacks) > 0 ) {
        fmt.Println("Active resources (" + region + "): " + strings.Join(stacks,", ") + "\n")
        handler := func(msg string) {
          attendant.Spinner().Stop()
          fmt.Println(msg)
          attendant.Spinner().Start()
        }
        dryrun, _ := cmd.Flags().GetBool("dry-run")
        attendant.SpinWithSuffix(func() { err = attendant.CleanFlightEventHandling(stacks, dryrun, handler) }, region)
        if err != nil { return err }
        fmt.Println("")
      }
    }
    return nil
  },
}

func init() {
  RootCmd.AddCommand(cleanupCmd)
  cleanupCmd.Flags().Bool("dry-run", false, "Perform a dry run displaying what resources would be cleaned")
  cleanupCmd.Flags().String("regions", "", "Select regions to query")
}
