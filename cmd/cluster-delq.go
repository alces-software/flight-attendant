// Copyright © 2017 Alces Software Ltd <support@alces-software.com>
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
var clusterDelqCmd = &cobra.Command{
  Use:   "delq <cluster> <name>",
  Short: "Remove a compute queue from a running Flight Compute cluster",
  Long: `Remove a compute queue from a running Flight Compute cluster.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    if len(args) <= 1 {
      cmd.Help()
      return nil
    }

    var domain *attendant.Domain
    var err error

    if err := attendant.PreflightCheck(); err != nil { return err }
    domain, err = findDomain("clusterDelq", false)
    if err != nil { return err }

    fmt.Printf("Removing queue '%s' from cluster '%s' in domain '%s' (%s)...\n\n", args[1], args[0], domain.Name, attendant.Config().AwsRegion)
    err = delq(domain, args[0], args[1])
    if err != nil { return err }
    fmt.Println("\nCluster queue destroyed.\n")
    return nil
  },
}

func init() {
  clusterCmd.AddCommand(clusterDelqCmd)
  addDomainFlag(clusterDelqCmd, "clusterDelq")
}

func delq(domain *attendant.Domain, clusterName, queueName string) error {
  handler, err := attendant.CreateDestroyHandler(attendant.ComputeGroupResourceCount)
  if err != nil { return err }
  cluster := attendant.NewCluster(clusterName, domain, handler)
  attendant.Spin(func() { err = cluster.DestroyQueue(queueName) })
  cluster.MessageHandler = nil
  return err
}
