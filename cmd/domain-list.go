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

// listCmd represents the list command
var domainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your Flight Compute domains",
	Long: `List your Flight Compute domains.`,
	Run: func(cmd *cobra.Command, args []string) {
    var domains []attendant.Domain
    var err error

    regions := getRegions(cmd)
    for _, region := range regions {
      attendant.Config().AwsRegion = region
      attendant.Spin(func() {
        domains, err = attendant.AllDomains()
      })
      if err != nil {
        fmt.Println(err.Error())
        return
      }
      fmt.Printf("== Domains (%s) ==\n", attendant.Config().AwsRegion)
      if len(domains) > 0 {
        for _, domain := range domains {
          if *domain.Stack.StackStatus == "CREATE_IN_PROGRESS" {
            fmt.Println(domain.Name + " (not ready)")
          } else {
            fmt.Println(domain.Name)
          }
        }
      } else {
        fmt.Println("<none>")
      }
      fmt.Println("")
    }
	},
}

func init() {
	domainCmd.AddCommand(domainListCmd)
  domainListCmd.Flags().String("regions", "", "Select regions to query")
}
