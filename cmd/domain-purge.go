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

  "github.com/alces-software/flight-attendant/attendant"
)

// purgeCmd represents the purge command
var domainPurgeCmd = &cobra.Command{
  Use:   "purge <domain>",
  Short: "Empty a Flight Compute domain of clusters and infrastructure",
  Long: `Empty a Flight Compute domain of clusters and infrastructure.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    var err error
    var status *attendant.DomainStatus

    if len(args) == 0 {
      cmd.Help()
      return nil
    }

    domain := attendant.NewDomain(args[0], nil)

    if confirmed, _ := cmd.Flags().GetBool("yes"); !confirmed {
      fmt.Printf("You must supply `--yes` parameter to confirm you want to purge domain: %s\n", domain.Name)
      return nil
    }

    attendant.Spin(func() { status, err = domain.Status() })
    if err != nil { return err }

    if len(status.Clusters) + len(status.Appliances) > 0 {
      var ch chan string = make(chan string)
      handler, err := attendant.CreateDestroyHandler(0)
      if err != nil { return err }
      for _, cluster := range status.Clusters {
        cluster.MessageHandler = handler
        go func(cluster *attendant.Cluster, ch chan<- string) {
          n := fmt.Sprintf("flight-%s-%s", cluster.Domain.Name, cluster.Name)
          handler("DELETE_IN_PROGRESS " + n + " " + n)
          cluster.Purge()
          ch <- n
        }(cluster, ch)
      }
      for _, appliance := range status.Appliances {
        appliance.MessageHandler = handler
        go func(appliance *attendant.Appliance, ch chan<- string) {
          handler("DELETE_IN_PROGRESS " + *appliance.Stack.StackName + " " + *appliance.Stack.StackName)
          appliance.Purge()
          ch <- *appliance.Stack.StackName
        }(appliance, ch)
      }
      count := len(status.Clusters) + len(status.Appliances)
      for item := <- ch; item != ""; item = <- ch {
        handler("DELETE_COMPLETE " + item + " " + item)
        count -= 1
        if count == 0 {
          break
        }
      }
      handler("DONE")
      fmt.Println("Purge complete.")
    } else {
      return fmt.Errorf("Domain '%s' (%s) has no running infrastructure or cluster stacks. Can't purge.\n", domain.Name, attendant.Config().AwsRegion)
    }

    return nil
  },
}

func init() {
  domainCmd.AddCommand(domainPurgeCmd)
  domainPurgeCmd.Flags().Bool("yes", false, "Confirm this dangerous operation")
}
