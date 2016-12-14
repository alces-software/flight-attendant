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
var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List running Flight Compute clusters",
	Long: `List running Flight Compute clusters.`,
	Run: func(cmd *cobra.Command, args []string) {
    var status *attendant.DomainStatus
    var err error

    solo, _ := cmd.Flags().GetBool("solo")
    if solo {
      attendant.Spin(func() { status, err = attendant.SoloStatus() })
    } else {
      domain, err := findDomain("clusterList", true)
      if err != nil {
        fmt.Println(err.Error())
        return
      }
      attendant.Spin(func() { status, err = domain.Status() })
    }

		if err != nil {
			fmt.Println(err.Error())
			return
		}

    if solo {
      fmt.Println("== Solo Clusters ==\n")
    } else {
      fmt.Println("== Clusters ==\n")
    }
    if len(status.Clusters) > 0 {
      for _, cluster := range status.Clusters {
        fmt.Println("    " + cluster.Name)
        fmt.Println("    " + strings.Repeat("-", len(cluster.Name)))
        for _, s := range strings.Split(cluster.GetAccessDetails(),"\n") {
          fmt.Println("    " + s)
        }
      }
    } else {
      fmt.Println("<none>")
    }
	},
}

func init() {
	clusterCmd.AddCommand(clusterListCmd)
  addDomainFlag(clusterListCmd, "clusterList")
  clusterListCmd.Flags().BoolP("solo", "s", false, "List Flight Compute Solo clusters")
}
