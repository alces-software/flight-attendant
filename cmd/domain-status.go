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
  "gopkg.in/yaml.v2"
)

var forceDomainStatus bool

// statusCmd represents the status command
var domainStatusCmd = &cobra.Command{
  Use:   "status <domain>",
  Short: "Show status of a Flight Compute domain",
  Long: `Show status a Flight Compute domain.`,
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
    all, _ := cmd.Flags().GetBool("all")
    showVpnConfig, _ := cmd.Flags().GetBool("show-vpn-config")

    if len(args) == 0 && (!all || showVpnConfig) {
      cmd.Help()
      return nil
    }

    if err := attendant.PreflightCheck(); err != nil { return err }
    if all {
      regions := getRegions(cmd)
      for _, region := range regions {
        var err error
        var domains []attendant.Domain
        attendant.Config().AwsRegion = region
        attendant.SpinWithSuffix(func() { domains, err = attendant.AllDomains() }, region)
        if err != nil { return err }
        for _, domain := range domains {
          statusFor(&domain)
          fmt.Println("")
        }
      }
    } else {
      domain := attendant.NewDomain(args[0], nil)
      if showVpnConfig {
        vpnConfigFor(domain)
      } else {
        if attendant.Config().SimpleOutput {
          simpleStatusFor(domain)
        } else {
          statusFor(domain)
        }
      }
    }
    return nil
  },
}

func init() {
  domainCmd.AddCommand(domainStatusCmd)
  domainStatusCmd.Flags().BoolP("all", "a", false, "Show all domains")
  domainStatusCmd.Flags().Bool("show-vpn-config", false, "Display VPN configuration details in a YAML format")
  domainStatusCmd.Flags().String("regions", "", "Select regions to query")
}

func vpnConfigFor(domain *attendant.Domain) {
  var err error
  var status *attendant.DomainStatus

  attendant.SpinWithSuffix(func() { status, err = domain.Status() }, attendant.Config().AwsRegion + ": " + domain.Name)
  if err != nil {
    fmt.Println(err.Error())
    return
  }
  if yaml, err := yaml.Marshal(&status.VPNDetails); err == nil {
    fmt.Println(string(yaml))
  }
}

func simpleStatusFor(domain *attendant.Domain) {
  status, err := domain.Status()
  if err != nil {
    fmt.Println(err.Error())
    return
  }
  if yaml, err := yaml.Marshal(status.Details()); err == nil {
    fmt.Println(string(yaml))
  }
}

func statusFor(domain *attendant.Domain) {
  var err error
  var status *attendant.DomainStatus

  attendant.SpinWithSuffix(func() { status, err = domain.Status() }, attendant.Config().AwsRegion + ": " + domain.Name)
  if err != nil {
    fmt.Println(err.Error())
    return
  }

  fmt.Printf(">>> Domain '%s' (%s) <<<\n\n", domain.Name, attendant.Config().AwsRegion)

  fmt.Println("== Networking ==")
  if status.HasInternetAccess {
    fmt.Println(" * Internet access: enabled")
  } else {
    fmt.Println(" * Internet access: disabled")
  }

  if status.VPNConnectionId != "" {
    fmt.Println(" * VPN connection: " + status.VPNConnectionId)
    fmt.Println("   Outside address: " + status.VPNDetails.OutsideClientAddr)
    fmt.Println("   Client ASN: " + status.VPNDetails.ClientASN)
    fmt.Println("   Tun 1 Client inside address: " + status.VPNDetails.Tunnel1.InsideClientAddr)
    fmt.Println("         AWS outside address: " + status.VPNDetails.Tunnel1.OutsideAwsAddr)
    fmt.Println("         AWS inside address: " + status.VPNDetails.Tunnel1.InsideAwsAddr)
    fmt.Println("         AWS ASN: " + status.VPNDetails.Tunnel1.AwsASN)
    fmt.Println("         Shared key: " + status.VPNDetails.Tunnel1.SharedKey)
    fmt.Println("   Tun 2 Client inside address: " + status.VPNDetails.Tunnel2.InsideClientAddr)
    fmt.Println("         AWS outside address: " + status.VPNDetails.Tunnel2.OutsideAwsAddr)
    fmt.Println("         AWS inside address: " + status.VPNDetails.Tunnel2.InsideAwsAddr)
    fmt.Println("         AWS ASN: " + status.VPNDetails.Tunnel2.AwsASN)
    fmt.Println("         Shared key: " + status.VPNDetails.Tunnel2.SharedKey)
  }

  if status.PeerVPC != "" {
    fmt.Println(" * Peer VPC: " + status.PeerVPC)
    fmt.Println(" * Peer network: " + status.PeerVPCCIDRBlock)
  }

  fmt.Println("\n== Infrastructure ==\n")
  if len(status.Appliances) > 0 {
    for _, appliance := range status.Appliances {
      fmt.Println("    " + appliance.Name)
      fmt.Println("    " + strings.Repeat("-", len(appliance.Name)))
      for _, s := range strings.Split(appliance.GetDetails(),"\n") {
        fmt.Println("    " + s)
      }
    }
  } else {
    fmt.Println("<none>\n")
  }

  fmt.Println("== Clusters ==\n")
  if len(status.Clusters) > 0 {
    for _, cluster := range status.Clusters {
      var details string
      attendant.SpinWithSuffix(func() { details = cluster.GetDetails() }, attendant.Config().AwsRegion + ": " + domain.Name + "/" + cluster.Name)
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
