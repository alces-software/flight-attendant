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
  "github.com/aws/aws-sdk-go/service/cloudformation"
  
  "github.com/alces-software/flight-attendant/attendant"
)

// launchCmd represents the launch command
var clusterListCmd = &cobra.Command{
  Use:   "list",
  Short: "List running Flight Compute clusters",
  Long: `List running Flight Compute clusters.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    var status *attendant.DomainStatus
    var domains []attendant.Domain
    var err error

    regions := getRegions(cmd)
    for _, region := range regions {
      attendant.Config().AwsRegion = region
      solo, _ := cmd.Flags().GetBool("solo")
      all, _ := cmd.Flags().GetBool("all")
      if !solo || all {
        if domain, err := findDomain("clusterList", false); err == nil {
          domains = []attendant.Domain{*domain}
        } else {
          if err.Error() == "This operation requires you to specify a domain" {
            attendant.SpinWithSuffix(func() {
              domains, err = attendant.AllDomains()
            }, region)
          } else { return err }
        }
        if err != nil { return err }
        for _, domain := range domains {
          attendant.SpinWithSuffix(func() { status, err = domain.Status() }, region + ": " + domain.Name)
          if err != nil { return err }
          if attendant.Config().SimpleOutput {
            for _, cluster := range status.Clusters {
              err = cluster.LoadComputeGroups()
              if err != nil { return err }
              fmt.Printf("%s=%d\n", cluster.Name, len(cluster.ComputeGroups))
            }
          } else {
            fmt.Printf("== Clusters in '%s' (%s) ==\n", domain.Name, attendant.Config().AwsRegion)
            printClusters(status)
            fmt.Println("")
          }
        }
      }
      if solo || all {
        attendant.SpinWithSuffix(func() { status, err = attendant.SoloStatus() }, region + " (Solo)")
        if err != nil { return err }
        if len(status.Clusters) > 0 {
          fmt.Printf("== Solo Clusters (%s) ==\n", attendant.Config().AwsRegion)
          printClusters(status)
          fmt.Println("")
        }
        if all {
          var others []*cloudformation.Stack
          attendant.SpinWithSuffix(func() { others, err = attendant.OtherStacks() }, region + " (Other resources)")
          if err != nil { return err }
          if len(others) > 0 {
            fmt.Printf("== Other resources (%s) ==\n", attendant.Config().AwsRegion)
            for _, stack := range others {
              guessType := ""
              for _, tag := range stack.Tags {
                if *tag.Key == "alces:orchestrator" {
                  guessType = " (Alces FlightDeck Resource)"
                  break
                }
              }
              if guessType == "" {
                if strings.Contains(*stack.Description, "Alces Flight Compute") {
                  guessType = " (Flight Compute from AWS Marketplace)"
                } else {
                  guessType = " (Unknown)"
                }
              }
              fmt.Println("    " + *stack.StackName + guessType)
            }
            fmt.Println("")
          }
        }
      }
    }
    return nil
  },
}

func init() {
  clusterCmd.AddCommand(clusterListCmd)
  addDomainFlag(clusterListCmd, "clusterList")
  clusterListCmd.Flags().BoolP("solo", "s", false, "List Flight Compute Solo clusters")
  clusterListCmd.Flags().BoolP("all", "a", false, "List Flight Compute Enterprise and Solo clusters")
  clusterListCmd.Flags().String("regions", "", "Select regions to query")
}

func printClusters(status *attendant.DomainStatus) {
  if len(status.Clusters) > 0 {
    for _, cluster := range status.Clusters {
      var details string
      clusterName := cluster.Name
      if cluster.Domain != nil {
        clusterName = cluster.Domain.Name + "/" + clusterName
      }
      attendant.SpinWithSuffix(func() { details = cluster.GetDetails() }, attendant.Config().AwsRegion + ": " + clusterName)
      fmt.Println("    " + cluster.Name)
      fmt.Println("    " + strings.Repeat("-", len(cluster.Name)))
      for _, s := range strings.Split(details,"\n") {
        fmt.Println("    " + s)
      }
    }
  } else {
    fmt.Println("<none>")
  }
}
