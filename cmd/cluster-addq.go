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
  "github.com/spf13/viper"
  
  "github.com/alces-software/flight-attendant/attendant"
)

// launchCmd represents the launch command
var clusterAddqCmd = &cobra.Command{
  Use:   "addq <cluster> <name>",
  Short: "Add a compute queue to a running Flight Compute cluster",
  Long: `Add a compute queue to a running Flight Compute cluster.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    if len(args) <= 1 {
      cmd.Help()
      return nil
    }

    computeInstanceType := viper.GetString("queue-instance-type")
    if computeInstanceType != "" {
      if ! attendant.IsValidComputeInstanceType(computeInstanceType) {
        return fmt.Errorf("Invalid compute instance type '%s'. Try one of: %s\n", computeInstanceType, attendant.ComputeInstanceTypes)
      }
    }

    var domain *attendant.Domain
    var err error

    domain, err = findDomain("clusterAddq", false)
    if err != nil { return err }

    componentParamsFile, err := cmd.Flags().GetString("params")
    if err != nil { return err }

    if err := setupTemplateSource("clusterAddq"); err != nil { return err }

    if err := setupKeyPair("clusterAddq"); err != nil { return err }

    fmt.Printf("Adding queue '%s' to cluster '%s' in domain '%s' (%s)...\n\n", args[1], args[0], domain.Name, attendant.Config().AwsRegion)
    err = addQ(domain, args[0], args[1], componentParamsFile)
    if err != nil { return err }
    fmt.Println("\nCluster queue created.\n")
    return nil
  },
}

func init() {
  clusterCmd.AddCommand(clusterAddqCmd)
  addDomainFlag(clusterAddqCmd, "clusterAddq")
  addKeyPairFlag(clusterAddqCmd, "clusterAddq")
  addTemplateSetFlag(clusterAddqCmd, "clusterAddq")
  clusterAddqCmd.Flags().StringP("params", "p", "", "File containing parameters to use for launching the queue")
  clusterAddqCmd.Flags().StringP("queue-instance-type", "t", "", "Compute instance type (default: \"" + attendant.ComputeInstanceTypes[0] + "\")")
  viper.BindPFlag("queue-instance-type", clusterAddqCmd.Flags().Lookup("queue-instance-type"))
}

func addQ(domain *attendant.Domain, clusterName, queueName, componentParamsFile string) error {
  handler, err := attendant.CreateCreateHandler(attendant.ComputeGroupResourceCount)
  if err != nil { return err }
  cluster := attendant.NewCluster(clusterName, domain, handler)
  if viper.GetString("compute-group-label") == "" {
    viper.Set("compute-group-label", queueName)
  }
  attendant.Spin(func() { err = cluster.AddQueue(queueName, componentParamsFile) })
  cluster.MessageHandler = nil
  return err
}
