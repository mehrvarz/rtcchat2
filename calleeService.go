// rtcchat2 calleeService.go
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
	"time"
)

var TAG3 = "CalleeService"
var webrootCallee = "html/callee"

var maxMakeKeys = 100
var MakeKeyMap = make(map[string]string, maxMakeKeys)

func CalleeService(secure bool, sigport int, callerPort int, autoAnswer string) {
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
			SigPort			int
			CallerPort		int
			SecureCallee	bool
			AutoAnswer		string
		}
		patchInfo := PatchInfo{sigport, callerPort, secure, autoAnswer}
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
		if r.URL.Path == "" || r.URL.Path == "/" || r.URL.Path == "/index.html" {
			// generate new random callee + caller keys (repeat while already available in rtcchat.gkv)
			callerKey, calleeKey := CreateNewKeys()
			keyKey := generateId()
			type PatchInfo struct {
				CalleeKey string
				CallerKey string
				KeyKey string
				SigPort int
				CallerPort int
				SecureCallee bool
			}
			patchInfo := PatchInfo{callerKey,calleeKey,keyKey,sigport,callerPort,secure}
			pathToIndex := "html/callee-new-keys" + "/index.html"
			fmt.Println(TAG, "patchInfo=", patchInfo, "serve=", pathToIndex)
			homeTempl := template.Must(template.ParseFiles(pathToIndex))
			homeTempl.Execute(w, patchInfo)
			MakeKeyMap[keyKey] = callerKey+","+calleeKey
			// user will activte these keys in: case "activateKeys"

			// TODO: what happens if MakeKeyMap runs full?

		} else if strings.HasPrefix(r.URL.Path, "/callee:") {
			calleeKey := r.RequestURI[8:]
			callerKey := getPersistedCallerKey(calleeKey)
			if(callerKey!="") {
				// hand over callerKey by patching callee/index.html
				// so that rtccallee.js can send back the callerKey via websocket "announce"
				// for CalleeMap[callerKey] = cws (see below)
				type PatchInfo struct {
					Key string
				}
				patchInfo := PatchInfo{callerKey} // callerKey = public-callee-key
				pathToIndex := webrootCallee + "/index.html"
				fmt.Println(TAG, "patchInfo=", patchInfo, "serve=", pathToIndex)
				homeTempl := template.Must(template.ParseFiles(pathToIndex))
				homeTempl.Execute(w, patchInfo)


			} else {
				// unknown callee key: show error page
				fmt.Println(TAG, "unknown calleeKey="+calleeKey+" show error page")
				// add artificial delay: fight user scanning
				select {
				case <-time.After(3 * time.Second):
					type PatchInfo struct {
						Key string
					}
					patchInfo := PatchInfo{calleeKey}
					htmlTempl := template.Must(template.ParseFiles("html/callee-unavailable/index.html"))
					htmlTempl.Execute(w, patchInfo)
				}
			}

		} else {
			redir := "/html" + r.RequestURI
			fmt.Println(TAG, "redir", redir)
			http.Redirect(w, r, redir, http.StatusMovedPermanently)
		}
	})

	localAddr := fmt.Sprintf(":%d", sigport+1)
	var err3 error = nil
	if secure {
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
	var callerKey = ""

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
			// set by rtccallee.js: announce callee's availability for incoming calls
			callerKey = msg["uniqueID"]
			CalleeMap[callerKey] = cws
			// callerService.go will find cws entry in CalleeMap[] (see: case "call":)
			// TODO: what if CalleeMap is full? Ideally the oldest entries would be removed

			fmt.Println(TAG3, "WsSessionHandlerCallee user with key is now registered:", callerKey)
			// TODO: this way we CANNOT have a callee be registered on multiple devices in parallel
			
			// this allows the callee to display it's caller-key as a link
			websocket.Message.Send(cws, fmt.Sprintf(`{"command":"callerKey","key": "%s"}`, callerKey))
			
			// TODO: if announced for the 1st time, we want to provide some 1st use explanation text to callee
			// check gkv for number of times used
			// but we only have the callerKey
			//msg := "you have been registered for caller id=" + callerKey
			//websocket.Message.Send(cws, fmt.Sprintf(`{"command":"info","msg": "%s"}`, msg))

		case "activateKeys":
			// sent by callee-new-keys.js
			keyKey := msg["key"]
			fmt.Println(TAG3, "WsSessionHandlerCallee user wants to activate key:", keyKey)
			bothKeys,ok := MakeKeyMap[keyKey]
			if(!ok) {
				fmt.Println(TAG3, "WsSessionHandlerCallee failed to activate:", keyKey)
				// send to callee-new-keys.js
				websocket.Message.Send(cws, `{"command":"activateConfirm","success": false}`)
			} else {
				//fmt.Println(TAG3, "WsSessionHandlerCallee ready to activate:", bothKeys)
				// store callee + caller keys in rtcchat.gkv
				strArray := strings.Split(bothKeys,",")
				fmt.Println(TAG3, "WsSessionHandlerCallee ready to activate:", strArray[0], strArray[1])
				StoreNewKeys(strArray[0], strArray[1])
				// TODO: how can StoreNewKeys() fail?
				delete(MakeKeyMap,keyKey)

				// send success to callee-new-keys.js
				websocket.Message.Send(cws, `{"command":"activateConfirm","success": true}`)
			}
		}
	}

	if callerKey != "" {
		// unregister callee
		delete(CalleeMap,callerKey)
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
