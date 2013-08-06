// rtcredirect
// Copyright 2013 Timur Mehrvarz. All rights reserved.
//
// #requirements:
// #go get labix.org/v2/mgo
// #run service: cd mongodb-linux-x86_64-2.4.5 
// #             mkdir ~/data/db
// #             bin/mongod --dbpath ~/data/db
// #announce:    https://(hostaddr):8000/(key)
// #redirect:    http://(hostaddr):8000/(key)
//
// callee:      https://(hostaddr):8078/callee:(callee-private-key)
// caller:      https://(hostaddr):8000/call:(callee-public-key)
// both will be redirected to:
//              https://localhost:8077/?room=(uniqueID)
//
//

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

	fmt.Println("secureRedirect",*secureRedirect)
	fmt.Println("sigport",*sigport)
	fmt.Println("callerport",*callerport)

	// StunUDP (default port 19253 = stunport)
	go rtcchat2.StunUDP(*stunhost,*stunport)

	// RtcSignaling (default port 8077)
	// makes use of "html/signaling/*" = sigport
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

