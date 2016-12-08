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
var clusterDestroyCmd = &cobra.Command{
	Use:   "destroy <cluster>",
	Short: "Destroy a running cluster",
	Long: `Destroy a running cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}

    domain, err := findDomain("clusterDestroy")
    if err != nil {
      fmt.Println(err.Error())
      return
    }

    fmt.Printf("Destroying cluster '%s' in domain '%s' (%s)...\n\n", args[0], domain.Name, attendant.Config().AwsRegion)
    err = destroyCluster(domain, args[0])
    if err != nil {
      fmt.Println(err.Error())
      return
    }
    fmt.Println("\nCluster destroyed.")
	},
}

func init() {
	clusterCmd.AddCommand(clusterDestroyCmd)
  addDomainFlag(clusterDestroyCmd, "clusterDestroy")
}

func destroyCluster(domain *attendant.Domain, name string) error {
  handler, err := attendant.CreateDestroyHandler(attendant.ClusterResourceCount)
  if err != nil { return err }
  cluster := attendant.NewCluster(name, domain, handler)
  attendant.Spin(func() { err = cluster.Destroy() })
  cluster.MessageHandler = nil
  return err
}
