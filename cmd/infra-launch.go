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
var infraLaunchCmd = &cobra.Command{
  Use:   "launch [<appliance>|--all|--base]",
  Short: "Launch a Flight infrastructure appliance",
  Long: `Launch a Flight infrastructure appliance.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    var err error

    all, _ := cmd.Flags().GetBool("all")
    base, _ := cmd.Flags().GetBool("base")
    if !all && !base {
      if len(args) == 0 {
        cmd.Help()
        return nil
      } else if ! attendant.IsValidApplianceType(args[0]) {
        return fmt.Errorf("Unknown appliance type: %s\n", args[0])
      }
    }

    if err := setupTemplateSource("infraLaunch"); err != nil { return err }
    if err := setupKeyPair("infraLaunch"); err != nil { return err }

    domain, err := findDomain("infraLaunch", true)
    if err != nil { return err }

    if base {
      for _, applianceName := range attendant.BaseApplianceNames {
        var appliance *attendant.Appliance
        fmt.Printf("Launching appliance '%s' in domain '%s' (%s)...\n\n", applianceName, domain.Name, attendant.Config().AwsRegion)
        appliance, err = launchAppliance(domain, applianceName)
        if err != nil { break }
        fmt.Println("\nAppliance launched.\n")
        fmt.Println("== Appliance details ==")
        fmt.Println(appliance.GetDetails() + "\n")
      }
    } else if all {
      for applianceName, _ := range attendant.ApplianceTemplates {
        var appliance *attendant.Appliance
        fmt.Printf("Launching appliance '%s' in domain '%s' (%s)...\n\n", applianceName, domain.Name, attendant.Config().AwsRegion)
        appliance, err = launchAppliance(domain, applianceName)
        if err != nil { break }
        fmt.Println("\nAppliance launched.\n")
        fmt.Println("== Appliance details ==")
        fmt.Println(appliance.GetDetails() + "\n")
      }
    } else {
      var appliance *attendant.Appliance
      fmt.Printf("Launching appliance '%s' in domain '%s' (%s)...\n\n", args[0], domain.Name, attendant.Config().AwsRegion)
      appliance, err = launchAppliance(domain, args[0])
      if err == nil {
        fmt.Println("\nAppliance launched.\n")
        fmt.Println("== Appliance details ==")
        fmt.Println(appliance.GetDetails() + "\n")
      }
    }
    if err != nil { return err }
    return nil
  },
}

func init() {
  infraCmd.AddCommand(infraLaunchCmd)
  addDomainFlag(infraLaunchCmd, "infraLaunch")
  addKeyPairFlag(infraLaunchCmd, "infraLaunch")
  addTemplateSetFlag(infraLaunchCmd, "infraLaunch")
  addTemplateRootFlag(infraLaunchCmd, "infraLaunch")

  infraLaunchCmd.Flags().StringP("instance-type", "i", "", fmt.Sprintf("Appliance instance type (default: %s)", attendant.ApplianceInstanceTypes[0]))
  viper.BindPFlag("appliance-instance-type", infraLaunchCmd.Flags().Lookup("instance-type"))

  infraLaunchCmd.Flags().BoolP("base", "b", false, "Launch all base appliances into a domain")
  infraLaunchCmd.Flags().BoolP("all", "a", false, "Launch all base and optional appliances into a domain")
}

func launchAppliance(domain *attendant.Domain, name string) (*attendant.Appliance, error) {
  instanceType := viper.GetString(name + "-instance-type")
  if instanceType == "" { instanceType = viper.GetString("appliance-instance-type") }
  if instanceType != "" && ! attendant.IsValidApplianceInstanceType(instanceType) {
    return nil, fmt.Errorf("Invalid instance type '%s'. Try one of: %s\n", instanceType, attendant.ApplianceInstanceTypes)
  }

  handler, err := attendant.CreateCreateHandler(attendant.ApplianceResourceCounts[name])
  if err != nil { return nil, err }
  appliance := attendant.NewAppliance(name, domain, handler)
  attendant.Spin(func() { err = appliance.Create() })
  appliance.MessageHandler = nil
  return appliance, err
}
