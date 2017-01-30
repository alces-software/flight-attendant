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

// launchCmd represents the launch command
var clusterReduceCmd = &cobra.Command{
  Use:   "reduce <cluster> <component>",
  Short: "Reduce running Flight Compute clusters",
  Long: `Reduce running Flight Compute clusters.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    if len(args) <= 1 {
      cmd.Help()
      return nil
    }

    var domain *attendant.Domain
    var err error

    domain, err = findDomain("clusterReduce", false)
    if err != nil { return err }

    componentName, err := cmd.Flags().GetString("name")
    if err != nil { return err }

    if componentName == "" {
      fmt.Printf("Reducing cluster '%s' in domain '%s' (%s) with '%s'...\n\n", args[0], domain.Name, attendant.Config().AwsRegion, args[1])
    } else {
      fmt.Printf("Reducing cluster '%s' in domain '%s' (%s) with '%s (%s)'...\n\n", args[0], domain.Name, attendant.Config().AwsRegion, args[1], componentName)
    }
    err = reduceCluster(domain, args[0], args[1], componentName)
    if err != nil { return err }
    fmt.Println("\nCluster reduced.\n")
    return nil
  },
}

func init() {
  clusterCmd.AddCommand(clusterReduceCmd)
  addDomainFlag(clusterReduceCmd, "clusterReduce")
  clusterReduceCmd.Flags().StringP("name", "n", "", "Provide the name of the component")
}

func reduceCluster(domain *attendant.Domain, clusterName, componentType, componentName string) error {
  handler, err := attendant.CreateDestroyHandler(0)
  if err != nil { return err }
  cluster := attendant.NewCluster(clusterName, domain, handler)
  attendant.Spin(func() { err = cluster.Reduce(componentType, componentName) })
  cluster.MessageHandler = nil
  return err
}
