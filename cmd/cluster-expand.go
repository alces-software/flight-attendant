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
var clusterExpandCmd = &cobra.Command{
	Use:   "expand <cluster> <component>",
	Short: "Expand running Flight Compute clusters",
	Long: `Expand running Flight Compute clusters.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) <= 1 {
			cmd.Help()
			return
		}

    var domain *attendant.Domain
    var err error

    domain, err = findDomain("clusterExpand", false)
    if err != nil {
      fmt.Println(err.Error())
      return
    }

    componentName, err := cmd.Flags().GetString("name")
    if err != nil {
      fmt.Println(err.Error())
      return
    }
    componentParamsFile, err := cmd.Flags().GetString("params")
    if err != nil {
      fmt.Println(err.Error())
      return
    }

    if err := setupTemplateSource("clusterExpand"); err != nil {
      fmt.Println(err.Error())
      return
    }

    if err := setupKeyPair("clusterExpand"); err != nil {
      fmt.Println(err.Error())
      return
    }

    if componentName == "" {
      fmt.Printf("Expanding cluster '%s' in domain '%s' (%s) with '%s'...\n\n", args[0], domain.Name, attendant.Config().AwsRegion, args[1])
    } else {
      fmt.Printf("Expanding cluster '%s' in domain '%s' (%s) with '%s (%s)'...\n\n", args[0], domain.Name, attendant.Config().AwsRegion, args[1], componentName)
    }
    err = expandCluster(domain, args[0], args[1], componentName, componentParamsFile)
    if err != nil {
      fmt.Println(err.Error())
      return
    }
    fmt.Println("\nCluster expanded.\n")
	},
}

func init() {
	clusterCmd.AddCommand(clusterExpandCmd)
  addDomainFlag(clusterExpandCmd, "clusterExpand")
  addKeyPairFlag(clusterExpandCmd, "clusterExpand")
  addTemplateSetFlag(clusterExpandCmd, "clusterExpand")
  clusterExpandCmd.Flags().StringP("name", "n", "", "Provide a name for the component")
  clusterExpandCmd.Flags().StringP("params", "p", "", "File containing parameters to use for launching the component")
}

func expandCluster(domain *attendant.Domain, clusterName, componentType, componentName, componentParamsFile string) error {
  handler, err := attendant.CreateCreateHandler(0)
  if err != nil { return err }
  cluster := attendant.NewCluster(clusterName, domain, handler)
  attendant.Spin(func() { err = cluster.Expand(componentType, componentName, componentParamsFile) })
  cluster.MessageHandler = nil
  return err
}
