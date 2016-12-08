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

package attendant

import (
  "fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/guregu/dynamo"
)

type DomainEntity struct {
  Name string `dynamo:",hash"`
  Prefix string
  NetBookings []int `dynamo:",set"`
}

type ClusterEntity struct {
  Name string `dynamo:",hash"`
  Domain string
  GroupCount int
  NetworkIndex int
}

func (d *Domain) SaveEntity() error {
  table, err := getTable("FlightDomains")
  if err != nil { return err }

  record := DomainEntity{Name: d.Name, Prefix: d.Prefix()}
  return table.Put(record).Run()
}

func (d *Domain) DestroyEntity() error {
  table, err := getTable("FlightDomains")
  if err != nil { return err }

  return table.Delete("Name", d.Name).Run()
}

func (d *Domain) LoadEntity() (*DomainEntity, error) {
  table, err := getTable("FlightDomains")
  if err != nil { return nil, err }

  var record DomainEntity
  err = table.Get("Name", d.Name).One(&record)

  return &record, err
}

func (d *Domain) BookNetwork() (int, error) {
  record, err := d.LoadEntity()
  if err != nil { return 0, err }

  for i := 0; i < 128; i++ {
    if !containsI(record.NetBookings, i) {
      table, err := getTable("FlightDomains")
      if err != nil { return 0, err }
      err = table.
        Update("Name", d.Name).
        AddIntsToSet("NetBookings", i).
        If("NOT contains(NetBookings, ?)",i).
        Run()
      if err != nil { return 0, err }
      return i, nil
    }
  }
  return 0, fmt.Errorf("No available networks.")
}

func (d *Domain) ReleaseNetwork(index int) error {
  record, err := d.LoadEntity()
  if err != nil { return err }

  if containsI(record.NetBookings, index) {
    table, err := getTable("FlightDomains")
    if err != nil { return err }

    err = table.
      Update("Name", d.Name).
      DeleteIntsFromSet("NetBookings", index).
      If("contains(NetBookings, ?)", index).
      Run()
    if err != nil { return err }
    return nil
  }
  return fmt.Errorf("Network not booked.")
}

func (c *Cluster) CreateEntity() error {
  table, err := getTable("FlightClusters")
  if err != nil { return err }

  record := ClusterEntity{Name: c.Name, Domain: c.Domain.Name, NetworkIndex: c.Network.Index, GroupCount: 1}
  return table.Put(record).Run()
}

func (c *Cluster) DestroyEntity() error {
  table, err := getTable("FlightClusters")
  if err != nil { return err }

  return table.Delete("Name", c.Name).Run()
}

func (d *Cluster) LoadEntity() (*ClusterEntity, error) {
  table, err := getTable("FlightClusters")
  if err != nil { return nil, err }

  var record ClusterEntity
  err = table.Get("Name", d.Name).One(&record)

  return &record, err
}

func getTable(tableName string) (*dynamo.Table, error) {
  db, err := Dynamo()
  switch tableName {
  case "FlightDomains":
    err = db.CreateTable("FlightDomains", DomainEntity{}).Run()
  case "FlightClusters":
    err = db.CreateTable("FlightClusters", ClusterEntity{}).Run()
  }    
  if aerr, ok := err.(awserr.Error); ok {
    switch aerr.Code() {
    case "ResourceInUseException":
      // Ok, table already created.
    default:
      return nil, err
    }
  }
  table := db.Table(tableName)
  return &table, nil
}
