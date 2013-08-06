// rtcredirect calleeService.go
// Copyright 2013 Timur Mehrvarz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rtcchat2

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"text/template"
)

var TAG3 = "CalleeService"
var webrootCallee = "html/callee"

func CalleeService(secure bool, sigport int) {
	fmt.Println(TAG3, "start...")
    calleeServeMux := http.NewServeMux()

	// handle serving the rtcchat.js template
	templFile := "/rtccallee.js"
	calleeServeMux.HandleFunc(templFile, func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(TAG3, "serve template", r.URL.Path)
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		// patch sigport and secure-flag into rtccallee.js
		// for performance: if rtccallee.js is not expected to be modified at runtime,
		// we can move the next two lines above, outside of HandleFunc()
		templFilePath := fmt.Sprintf("%s%s", webrootCallee, templFile)
		htmlTempl := template.Must(template.ParseFiles(templFilePath))
		type PatchInfo struct {
			SigPort  int
			SecureRedirect bool
		}
		patchInfo := PatchInfo{sigport,secure}
		htmlTempl.Execute(w, patchInfo)
	})

	// handle alive and annouce msgs from rtccallee.js
	calleeServeMux.Handle("/ws", websocket.Handler(WsHandlerCallee))

	// handle serving of static web content from the "html" folder
	calleeServeMux.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))

	calleeServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "http method not allowed", 405)
			return
		}

		//fmt.Println(TAG, "serve r.URL.Path", r.URL.Path)
		if(r.URL.Path=="" || r.URL.Path=="/" || r.URL.Path=="/index.html") {
			redir := "/"+webrootCallee+"/index.html"
			// TODO: here we want to offer a tool to generate private + public URL's for the callee			
			fmt.Println(TAG,"redir", redir," *** MUST OFFER TOOL TO GENERATE URL's ***")
			http.Redirect(w,r,redir,http.StatusMovedPermanently)

		} else if(strings.HasPrefix(r.URL.Path,"/callee:")) {
			privatKey := r.RequestURI[8:]
			key := privatKey

			// TODO: need to convert from private-callee-key to public-callee-key (using mgo)
			switch(privatKey) {
			    case "f3e2-01d2-1f66-0cd3":     // tm
        			key = "4a-5g-9a-21"
			    case "01d2-f3e2-d332-164d":     // ulrich
        			key = "52-14-a4-67"
			    case "e6a2-dde2-d1f2-1622":     // tm for usbrom n7
        			key = "12-4d-c4-16"
			    case "d9a6-c4e4-2231-8a3d":     // nick
        			key = "23-54-d5-2c"
            }

/*
			// now we check if the public key is marked as being online
			_, ok := CalleeMap[key]
			if !ok {
				// unknown key: respond html "Requested user is currently not available"
				fmt.Println(TAG, "unknown key", key)
				// add artificial delay: fight user scanning
				select {
				case <-time.After(3 * time.Second):
					type PatchInfo struct {
						Key   string
					}
					patchInfo := PatchInfo{key}
					htmlTempl := template.Must(template.ParseFiles("html/callee-unavailable/index.html"))
					htmlTempl.Execute(w, patchInfo)
				}
				return
			}
*/
			// requested client is online
			fmt.Println(TAG, "user with key is online:", key)

			// open html-form to allow caller to enter his own name
			type PatchInfo struct {
				Key string
			}
			patchInfo := PatchInfo{key} // key = public-callee-key
			pathToIndex := webrootCallee+"/index.html"
			fmt.Println(TAG,"patchInfo=", patchInfo, "serve=",pathToIndex)
			homeTempl := template.Must(template.ParseFiles(pathToIndex))
			homeTempl.Execute(w, patchInfo)

		} else {
			redir := "/html"+r.RequestURI
			fmt.Println(TAG,"redir", redir)
			http.Redirect(w,r,redir,http.StatusMovedPermanently)
		}
	})

	localAddr := fmt.Sprintf(":%d", sigport+1)
	var err3 error = nil
	if(secure) {
		fmt.Println(TAG3, "ListenAndServeTLS", localAddr)
		err3 = http.ListenAndServeTLS(localAddr, certFile, keyFile, calleeServeMux)
	} else {
		fmt.Println(TAG3, "ListenAndServe", localAddr)
		err3 = http.ListenAndServe(localAddr, calleeServeMux)
	}
	if err3 != nil {
		fmt.Println(TAG3, "fatal error ", err3.Error())
		os.Exit(1)
	}
}

// handle all callee websockets sessions
func WsHandlerCallee(cws *websocket.Conn) {
	fmt.Println(TAG3, "WsHandlerCallee start new client session...")
	doneWsSessionHandlerCallee := make(chan bool)
	go WsSessionHandlerCallee(cws, doneWsSessionHandlerCallee)
	<-doneWsSessionHandlerCallee
	fmt.Println(TAG3, "WsHandlerCallee done")
}

// handle one complete websockets session
func WsSessionHandlerCallee(cws *websocket.Conn, doneWsSessionHandlerCallee chan bool) {
	err := websocket.Message.Send(cws, `{"command":"alive!"}`)
	if err != nil {
		fmt.Println(TAG3, "WsSessionHandlerCallee failed to send 'connect' state", err)
		doneWsSessionHandlerCallee <- true
		return
	}

	quit := false
	for !quit {
		//fmt.Println(TAG3,"WsSessionHandlerCallee waiting for command from client...")
		var msg map[string]string
		err := websocket.JSON.Receive(cws, &msg)
		if err != nil {
			if err == io.EOF {
				fmt.Println(TAG3, "WsSessionHandlerCallee received EOF")
			} else {
				fmt.Println(TAG3, "WsSessionHandlerCallee can't receive", err)
			}
			break
		}

		switch msg["command"] {
		case "alive?":
			err := websocket.Message.Send(cws, `{"command":"alive!"}`)
			if err != nil {
				fmt.Println(TAG3, "WsSessionHandlerCallee failed to send 'alive' response", err)
				quit = true
				break
			}

		case "announce":
			uniqueID := msg["uniqueID"]
			CalleeMap[uniqueID] = cws
			fmt.Println(TAG3, "WsSessionHandlerCallee user with key is now registered:",uniqueID)
			msg := "you have been registered with uniqueID="+uniqueID
			websocket.Message.Send(cws, fmt.Sprintf(`{"command":"info","msg": "%s"}`, msg))
/*
		case "stopRing":
			fmt.Println(TAG3, "WsSessionHandlerCallee stopRing")
			calleekey := msg["calleekey"]
			fmt.Println(TAG3, "WsSessionHandlerCallee stopRing calleekey=", calleekey)
			if calleekey != "" {
				calleeCws, ok := CalleeMap[calleekey]
				if ok {
					fmt.Println(TAG3, "WsSessionHandlerCallee stopRing websocket.Message.Send()")
					websocket.Message.Send(calleeCws, `{"command":"stopRing"}`)
				}
			}
*/
		}
	}

	fmt.Println(TAG3, "WsSessionHandlerCallee done")
	doneWsSessionHandlerCallee <- true
}

// unique id generator
func generateId() string {
	var S4 = func() string {
		return fmt.Sprintf("%x", ((1 + rand.Intn(0x10000)) | 0))[1:]
	}
	return (S4() + S4() + "-" + S4() + "-" + S4() + "-" + S4() + "-" + S4() + S4() + S4())
}

