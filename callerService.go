// rtcchat2 callerService.go
// Copyright 2013 Timur Mehrvarz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rtcchat2

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

var TAG = "CallerService"
var certFile = "keys/cert.pem"
var keyFile = "keys/key.pem"

// CalleeMap maps the public callee-key to the calle-websocketConn-handle
var maxCallees = 1000
var CalleeMap = make(map[string]*websocket.Conn, maxCallees)

func ckeckKeys() {
	// make sure our https-keys are available
	_, err1 := os.Stat(certFile)
	if err1 != nil {
		fmt.Println(TAG, "missing", certFile)
		os.Exit(1)
	}
	_, err2 := os.Stat(keyFile)
	if err2 != nil {
		fmt.Println(TAG, "missing", keyFile)
		os.Exit(1)
	}
}

func CallerService(secure bool, callerport int) {
	callerServeMux := http.NewServeMux()

	fmt.Println(TAG, "start...")
	if secure {
		ckeckKeys()
	}

	// handle signaling (matching two clients into one room by secret word)
	callerServeMux.Handle("/ws", websocket.Handler(WsHandlerCaller))

	callerServeMux.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))

	callerServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "http method not allowed", 405)
			return
		}

		//fmt.Println(TAG, "serve r.URL.Path", r.URL.Path)
		if strings.HasPrefix(r.URL.Path, "/call:") {
			key := r.RequestURI[6:]
			_, ok := CalleeMap[key]
			if !ok {
				// unknown key: respond html "Requested user is currently not available"
				fmt.Println(TAG, "unknown key", key)
				// add artificial delay: fight user scanning
				select {
				case <-time.After(3 * time.Second):
					type PatchInfo struct {
						Key string
					}
					patchInfo := PatchInfo{key}
					htmlTempl := template.Must(template.ParseFiles("html/callee-unavailable/index.html"))
					htmlTempl.Execute(w, patchInfo)
				}
				return
			}

			// requested client is online
			fmt.Println(TAG, "user with key is online:", key)
			// open html-form to allow caller to enter his own name
			type PatchInfo struct {
				Key   string
				Title string
			}
			patchInfo := PatchInfo{key, "Calling rtc chat user: " + key}
			fmt.Println(TAG, "patchInfo", patchInfo, "serve 'html/caller-enter-name/index.html' ...")
			homeTempl := template.Must(template.ParseFiles("html/caller-enter-name/index.html"))
			homeTempl.Execute(w, patchInfo)

		} else {
			redir := "/html" + r.RequestURI
			fmt.Println(TAG, "redir", redir)
			http.Redirect(w, r, redir, http.StatusMovedPermanently)
		}
	})

	localAddr := fmt.Sprintf(":%d", callerport)
	var err2 error = nil
	if secure {
		fmt.Println(TAG, "ListenAndServeTLS", localAddr)
		err2 = http.ListenAndServeTLS(localAddr, certFile, keyFile, callerServeMux)
	} else {
		fmt.Println(TAG, "ListenAndServe", localAddr)
		err2 = http.ListenAndServe(localAddr, callerServeMux)
	}
	if err2 != nil {
		fmt.Println(TAG, "fatal error ", err2.Error())
		os.Exit(1)
	}

	fmt.Println(TAG, "service ready")
}

// handle all caller websockets sessions
func WsHandlerCaller(cws *websocket.Conn) {
	fmt.Println(TAG, "WsHandlerCaller start new client session...")
	doneWsSessionHandler := make(chan bool)
	go WsSessionHandlerCaller(cws, doneWsSessionHandler)
	<-doneWsSessionHandler
	fmt.Println(TAG, "WsHandlerCaller done")
}

// handle one complete websockets session
func WsSessionHandlerCaller(cws *websocket.Conn, doneWsSessionHandler chan bool) {
	quit := false
	for !quit {
		//fmt.Println(TAG,"WsSessionHandlerCaller waiting for command from client...")
		var msg map[string]string
		err := websocket.JSON.Receive(cws, &msg)
		if err != nil {
			if err == io.EOF {
				fmt.Println(TAG, "WsSessionHandlerCaller received EOF")
			} else {
				fmt.Println(TAG, "WsSessionHandlerCaller can't receive", err)
			}
			break
		}

		switch msg["command"] {
		case "alive?":
			err := websocket.Message.Send(cws, `{"command":"alive!"}`)
			if err != nil {
				fmt.Println(TAG3, "WsSessionHandlerCaller failed to send 'connect' state", err)
				quit = true
				break
			}

		case "call":
			// caller-enter-name.js is sending us: caller-username + callee-key
			callerName := msg["name"]
			calleeKey := msg["key"]
			linkType := msg["linktype"]
			fmt.Println(TAG, "call callerName=", callerName, " calleeKey=", calleeKey, " linkType", linkType)

			// get callee-cws via callee-key
			// TODO: it would be nice to support multiple calleeKey entries in CalleeMap[]
			// and send all of them the links below
			calleeCws, ok := CalleeMap[calleeKey]
			if !ok {
				fmt.Println(TAG, "key not found:", calleeKey)
				//http.Error(w, "key not found", 405)	// TODO?
				return
			}

			// notify callee (via cws) of incoming call from caller-username
			// this will offer the callee a link to answer the call
			// and it will make the callee's browser start ringing
			uniqueRoomName := generateId()
			err := websocket.Message.Send(calleeCws,
				fmt.Sprintf(`{"command":"newRoom","callerName": "%s", "roomName": "%s", "linkType": "%s"}`,
					callerName, uniqueRoomName, linkType))
			if err != nil {
				fmt.Println(TAG3, "WsSessionHandlerCaller connect: websocket.Message.Send err:", err)
			} else {
				fmt.Println(TAG3, "WsSessionHandlerCaller connect: websocket.Message.Send done")
				// callee now has a unique link to the signaling service session

				websocket.Message.Send(cws,
					fmt.Sprintf(`{"command":"newRoom", "roomName": "%s"}`, uniqueRoomName))

				// we can now end the caller websocket session
				quit = true
			}
		}
	}

	fmt.Println(TAG, "WsSessionHandlerCaller done")
	doneWsSessionHandler <- true
}
