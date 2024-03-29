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

// createCmd represents the create command
var domainCreateCmd = &cobra.Command{
  Use:   "create <domain>",
  Short: "Create a Flight Compute domain",
  Long: `Create a Flight Compute domain.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    if len(args) == 0 {
      cmd.Help()
      return nil
    }

    if err := setupTemplateSource("domainCreate"); err != nil { return err }

    domainParamsFile, err := cmd.Flags().GetString("params")
    if err != nil { return err }

    if err := attendant.PreflightCheck(); err != nil { return err }

    fmt.Printf("Creating domain '%s' (%s)...\n\n", args[0], attendant.Config().AwsRegion)
    _, err = createDomain(args[0], domainParamsFile)
    if err != nil { return err }

    fmt.Println("\nDomain created.")
    return nil
  },
}

func init() {
  domainCmd.AddCommand(domainCreateCmd)
  addTemplateSetFlag(domainCreateCmd, "domainCreate")
  addTemplateRootFlag(domainCreateCmd, "domainCreate")
  domainCreateCmd.Flags().StringP("params", "p", "", "File containing parameters to use when creating the domain")
}

func createDomain(name string, domainParamsFile string) (*attendant.Domain, error) {
  handler, err := attendant.CreateCreateHandler(attendant.DomainResourceCount)
  if err != nil { return nil, err }
  domain := attendant.NewDomain(name, handler)
  attendant.Spin(func() { err = domain.Create(name, domainParamsFile) } )
  domain.MessageHandler = nil
  return domain, err
}
