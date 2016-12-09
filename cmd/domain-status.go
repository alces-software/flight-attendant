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

var forceDomainStatus bool

// statusCmd represents the status command
var domainStatusCmd = &cobra.Command{
	Use:   "status <domain>",
	Short: "Show status of a Flight Compute domain",
	Long: `Show status a Flight Compute domain.`,
	Run: func(cmd *cobra.Command, args []string) {
    var status *attendant.DomainStatus
    var err error
    
		if len(args) == 0 {
			cmd.Help()
			return
		}

    domain := attendant.NewDomain(args[0], nil)
    attendant.Spin(func() { status, err = domain.Status() })
    if err != nil {
      fmt.Println(err.Error())
      return
    }

    fmt.Printf(">>> Domain '%s' (%s) <<<\n\n", domain.Name, attendant.Config().AwsRegion)
    
    fmt.Println("== Infrastructure ==")
    if len(status.Appliances) > 0 {
      for _, appliance := range status.Appliances {
        fmt.Println(appliance.Name)
      }
    } else {
      fmt.Println("<none>")
    }

    fmt.Println("\n== Clusters ==")
    if len(status.Clusters) > 0 {
      for _, cluster := range status.Clusters {
        fmt.Println(cluster.Name)
      }
    } else {
      fmt.Println("<none>")
    }
	},
}

func init() {
	domainCmd.AddCommand(domainStatusCmd)
}