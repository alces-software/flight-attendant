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

// destroyCmd represents the destroy command
var domainDestroyCmd = &cobra.Command{
  Use:   "destroy <domain>",
  Short: "Destroy a Flight Compute domain",
  Long: `Destroy a Flight Compute domain.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    var err error
    var status *attendant.DomainStatus

    if len(args) == 0 {
      cmd.Help()
      return nil
    }

    domain := attendant.NewDomain(args[0], nil)
    attendant.Spin(func() { status, err = domain.Status() })
    if err != nil { return err }

    if len(status.Clusters) + len(status.Appliances) > 0 {
      if force, _ := cmd.Flags().GetBool("force"); force {
        for _, cluster := range status.Clusters {
          fmt.Printf("Destroying cluster '%s' in domain '%s' (%s)...\n\n", cluster.Name, domain.Name, attendant.Config().AwsRegion)
          err = destroyCluster(domain, cluster.Name)
          if err != nil { return err }
          fmt.Println("\nCluster destroyed.\n")
        }
        for _, appliance := range status.Appliances {
          fmt.Printf("Destroying appliance '%s' in domain '%s' (%s)...\n\n", appliance.Name, domain.Name, attendant.Config().AwsRegion)
          err = destroyAppliance(domain, appliance.Name)
          if err != nil { return err }
          fmt.Println("\nAppliance destroyed.\n")
        }
      } else {
        return fmt.Errorf("Domain '%s' (%s) has running infrastructure or cluster stacks. Can't destroy.\n", domain.Name, attendant.Config().AwsRegion)
      }
    }

    fmt.Printf("Destroying domain '%s' (%s)...\n\n", domain.Name, attendant.Config().AwsRegion)
    err = destroyDomain(domain)
    if err != nil { return err }
    fmt.Println("Domain destroyed.")
    return nil
  },
}

func init() {
  domainCmd.AddCommand(domainDestroyCmd)
  domainDestroyCmd.Flags().BoolP("force", "f", false, "Destroy all clusters and infrastructure appliances along with the domain")
}

func destroyDomain(domain *attendant.Domain) error {
  // XXX - count should be determined based on whether the domain is peered or not
  handler, err := attendant.CreateDestroyHandler(attendant.BareDomainResourceCount)
  if err != nil { return err }
  domain.MessageHandler = handler
  attendant.Spin(func() { err = domain.Destroy() })
  domain.MessageHandler = nil
  return err
}
