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
var infraDestroyCmd = &cobra.Command{
  Use:   "destroy <appliance>",
  Short: "Destroy a running infrastructure appliance",
  Long: `Destroy a running infrastructure appliance.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    all, _ := cmd.Flags().GetBool("all")
    if !all {
      if len(args) == 0 {
        cmd.Help()
        return nil
      } else if ! attendant.IsValidApplianceType(args[0]) {
        return fmt.Errorf("Unknown appliance type: %s\n", args[0])
      }
    }

    if err := attendant.PreflightCheck(); err != nil { return err }
    domain, err := findDomain("infraDestroy", false)
    if err != nil { return err }

    var status *attendant.DomainStatus
    attendant.Spin(func() { status, err = domain.Status() })
    if err != nil { return err }

    force, _ := cmd.Flags().GetBool("force")

    if !force && len(status.Clusters) > 0 {
      return fmt.Errorf("Unable to destroy infrastructure appliance while domain '%s' has running clusters.\n", domain.Name)
    }

    if all {
      for appliance, _ := range status.Appliances {
        fmt.Printf("Destroying appliance '%s' in domain '%s' (%s)...\n\n", appliance, domain.Name, attendant.Config().AwsRegion)
        err = destroyAppliance(domain, appliance)
        if err != nil { break }
        fmt.Println("\nAppliance destroyed.")
      }
    } else {
      fmt.Printf("Destroying appliance '%s' in domain '%s' (%s)...\n\n", args[0], domain.Name, attendant.Config().AwsRegion)
      err = destroyAppliance(domain, args[0])
      if err == nil { fmt.Println("\nAppliance destroyed.") }
    }
    if err != nil { return err }
    return nil
  },
}

func init() {
  infraCmd.AddCommand(infraDestroyCmd)
  addDomainFlag(infraDestroyCmd, "infraDestroy")
  infraDestroyCmd.Flags().BoolP("all", "a", false, "Destroy all infrastructure appliances in the domain")
  infraDestroyCmd.Flags().BoolP("force", "f", false, "Destroy infrastructure appliance even if clusters are running in the domain")
}

func destroyAppliance(domain *attendant.Domain, name string) error {
  handler, err := attendant.CreateDestroyHandler(attendant.ApplianceResourceCounts[name])
  if err != nil { return err }
  appliance := attendant.NewAppliance(name, domain, handler)
  attendant.Spin(func() { err = appliance.Destroy() })
  appliance.MessageHandler = nil
  return err
}
