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
  "log"
  "os"
  "strconv"
  "strings"
  "sync"
  "time"

  "github.com/briandowns/spinner"
  "github.com/sethgrid/curse"
)

var attSpinner = spinner.New(spinner.CharSets[11], 100*time.Millisecond)  // Build our new spinner
var loggingEnabled = false

func Spinner() *spinner.Spinner {
  return attSpinner
}

func Spin(fn func()) {
  attSpinner.Start()
  fn()
  attSpinner.Stop()
}

func SpinWithSuffix(fn func(), suffix string) {
  attSpinner.Suffix = " " + suffix
  Spin(fn)
  attSpinner.Suffix = ""
}

func CreateCreateHandler(resourceTotal int) (func(msg string), error) {
  return createHandlerFunction(resourceTotal, "CREATE_IN_PROGRESS", "CREATE_COMPLETE", "✅")
}

func CreateDestroyHandler(resourceTotal int) (func(msg string), error) {
  return createHandlerFunction(resourceTotal, "DELETE_IN_PROGRESS", "DELETE_COMPLETE", "❎")
}

func createHandlerFunction(resourceTotal int, inProgressText, completeText, completionRune string) (func(msg string), error) {
  var resRegistry = make(map[string]int)
  var completeRegistry = make(map[string]bool)
  var disableCounters = false
  var counterDelta = 0
  var mutex sync.RWMutex

  if loggingEnabled {
    f, err := os.OpenFile("fly.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil { return nil, err }
    defer f.Close()
    log.SetOutput(f)
  }
  
  c, err := curse.New()
  if err != nil { return nil, err }

  fn := func(msg string) {
    mutex.Lock()
    defer mutex.Unlock()
    if loggingEnabled { log.Println(msg) }
    if msg == "DONE" {
      Spinner().Stop()
      for res, idx := range resRegistry {
        if ! completeRegistry[res] {
          lines := len(resRegistry) - idx
          c.MoveUp(lines).EraseCurrentLine()
          s := strings.Split(res, " ")
          fmt.Printf("%s  %s\n", completionRune, s[0])
          c.MoveDown(lines - 1)
        }
      }
      Spinner().Suffix = ""
      Spinner().Start()
      return
    } else if strings.HasPrefix(msg, "COUNTERS=") {
      s := strings.Split(msg, "=")
      c, err := strconv.Atoi(s[1])
      if err == nil { resourceTotal = c }
      return
    } else if msg == "DISABLE-COUNTERS" {
      disableCounters = true
      return
    } else if msg == "ENABLE-COUNTERS" {
      disableCounters = false
      return
    }
    s := strings.Split(strings.TrimSpace(msg), " ")
    state := s[0]
    res := s[1] + " " + s[2]
    name := s[1]
    if state != inProgressText && state != completeText {
      return
    } else if state == inProgressText {
      if _, exists := resRegistry[res]; !exists {
        Spinner().Stop()
        fmt.Println("⏳  " + name)
        Spinner().Start()
        resRegistry[res] = len(resRegistry)
      }
    } else {
      if completeRegistry[res] == true {
        return
      } else {
        Spinner().Stop()
        if _, exists := resRegistry[res]; !exists {
          fmt.Printf("%s  %s\n", completionRune, name)
          resRegistry[res] = len(resRegistry)
        } else {
          lines := len(resRegistry) - resRegistry[res]
          c.MoveUp(lines).EraseCurrentLine()
          fmt.Printf("%s  %s\n", completionRune, name)
          c.MoveDown(lines - 1)
          if disableCounters {
            counterDelta += 1
          }
        }
        completeRegistry[res] = true
        if resourceTotal > 0 && !disableCounters {
          Spinner().Suffix = fmt.Sprintf(" (%d/%d)", len(completeRegistry) - counterDelta, resourceTotal)
        }
        Spinner().Start()
      }
    }
  }
  return fn, nil
}
