// rtcchat2
// Copyright 2013 Timur Mehrvarz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"github.com/mehrvarz/rtcchat2"
	"os"
)

func main() {
	var secureRedirect = flag.Bool(  "secureRedirect", true, "set false to make redirect use http instead of https")
	var secureCallee   = flag.Bool(  "secureCallee", true, "set false to make callee use http instead of https")
	var sigport        = flag.Int(   "sigport", 8077, "set port for client signaling ")
	var callerport     = flag.Int(   "callerport", 8000, "set port for redirect service")
	var stunport       = flag.Int(   "stunport", 19253, "set STUN port")
	var stunhost       = flag.String("stunhost", "", "set STUN host redirect (empty=>weburl,non-empty=>no service)")
	var ringtone       = flag.String("ringtone", "", "set newroom ringtone (e.g. -newroom=sample/ring.ogg)")
	flag.Usage = func () {
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.Parse()

	fmt.Println("sigport",*sigport)
	fmt.Println("callerport",*callerport)
	fmt.Println("stunport",*stunport)

	// StunUDP (default port 19253 = stunport)
	go rtcchat2.StunUDP(*stunhost,*stunport)

	// RtcSignaling (default port 8077 = sigport)
	// makes use of "html/signaling/*"
	go rtcchat2.RtcSignaling(true,*sigport,*stunport,*stunhost,*ringtone)

	// CalleeService (default port 8078 = sigPort+1) allows a callee to wait for and receive chat calls
	// makes use of "html/callee/*"
	go rtcchat2.CalleeService(*secureCallee, *sigport)

	// CallerService (default port 8000 = callerport) allows other clients to "call" admin clients
	// makes use of "html/caller-enter-name/*" and "html/callee-unavailable/*"
	go rtcchat2.CallerService(*secureRedirect, *callerport)

	// let services run til aborted
	select {}
}

