// rtcchat2 rtcSignaling.go
// Copyright 2013 Timur Mehrvarz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// rtcSignaling implements a Websocket rendezvous server

package rtcchat2

import (
//	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	//"syscall"
	"text/template"
	"time"
	"golang.org/x/net/websocket"
)

var TAG2 = "RtcSignaling"
var ringtoneFile = ""

type roomInfo struct {
	clientId     string
	cws          *websocket.Conn
	linkType	 string
	users        int
	newRoomCmd   *exec.Cmd
	abortCmdChan chan bool
}

// max number of concurrently open rooms
var maxOpenRooms = 1000
var roomInfoMap = make(map[string]roomInfo, maxOpenRooms)
var maxAdminClients = 1000
var adminChannelsMap = make(map[string]chan string, maxAdminClients)
var sessionNumber = 0
var webroot = "html/signaling"

func RtcSignaling(secure bool, sigport int, stunport int, stunhost string, setRingtone string) {
	certFile := "keys/cert.pem"
	keyFile := "keys/key.pem"
	ringtoneFile = setRingtone

	if secure {
		// make sure our https-keys are there
		_, err1 := os.Stat(certFile)
		if err1 != nil {
			fmt.Println(TAG2, "missing", certFile)
			os.Exit(1)
		}

		_, err2 := os.Stat(keyFile)
		if err2 != nil {
			fmt.Println(TAG2, "missing", keyFile)
			os.Exit(1)
		}
	}

	// make sure random is really random (for generateId())
	rand.Seed(time.Now().UnixNano())

	// handle serving the rtcchat.js template
	templFile := "/rtcchat.js"
	http.HandleFunc(templFile, func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(TAG2, "serve template", r.URL.Path)
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		// patch sigport and stunport into rtcchat.js - if rtcchat.js is not modified
		// in production, move next two lines up above HandleFunc() for performance
		templFilePath := fmt.Sprintf("%s%s", webroot, templFile)
		homeTempl := template.Must(template.ParseFiles(templFilePath))

		type PatchInfo struct {
			SigPort  int
			StunPort int
			StunHost string
		}
		patchInfo := PatchInfo{sigport, stunport, stunhost}
		homeTempl.Execute(w, patchInfo)
	})

	// handle signaling (matching two clients into one room by secret word)
	http.Handle("/ws", websocket.Handler(WsHandler))

	// handle serving of static web content from the "static" folder
	http.Handle("/", http.FileServer(http.Dir(webroot)))

	localAddr := fmt.Sprintf(":%d", sigport)
	if secure {
		// start https server and listen for incoming requests using our defined handlers
		fmt.Println(TAG2, "ListenAndServeTLS", localAddr)
		err3 := http.ListenAndServeTLS(localAddr, certFile, keyFile, nil)
		if err3 != nil {
			fmt.Println(TAG2, "fatal error ", err3.Error())
			os.Exit(1)
		}
	} else {
		// start http server and listen for incoming requests using our defined handlers
		fmt.Println(TAG2, "ListenAndServe", localAddr)
		err3 := http.ListenAndServe(localAddr, nil)
		if err3 != nil {
			fmt.Println(TAG2, "fatal error ", err3.Error())
			os.Exit(1)
		}
	}
}

// handle all client websockets sessions
func WsHandler(cws *websocket.Conn) {
	fmt.Println(TAG2, "WsHandler start new client session...")
	done := make(chan bool)
	go WsSessionHandler(cws, done)
	<-done
}

// handle one complete websockets session
func WsSessionHandler(cws *websocket.Conn, done chan bool) {
	var myClientId string
	var roomName string
	var linkType string
	var otherCws *websocket.Conn = nil
	var forRingCws *websocket.Conn = nil
	var userNum = 0

	err := websocket.Message.Send(cws, `{"command":"connect"}`)
	if err != nil {
		fmt.Println(TAG2, userNum,"WsSessionHandler failed to send 'connect' state", err)
		done <- true
		return
	}

	var newRoomCmd *exec.Cmd = nil
	var abortCmdChan chan bool

	for {
		//fmt.Println(TAG2,userNum,"WsSessionHandler waiting for command from client...")
		var msg map[string]string
		err := websocket.JSON.Receive(cws, &msg)
		if err != nil {
			if err == io.EOF {
				fmt.Println(TAG2, userNum,"WsSessionHandler received EOF for myClientId=", myClientId)
				if otherCws != nil {
					// send presence=offline to otherCws
					websocket.Message.Send(otherCws, `{"command":"presence", "state":"offline"}`)
				}
			} else {
				fmt.Println(TAG2, userNum,"WsSessionHandler can't receive for myClientId=", myClientId, err)
			}
			break
		}

		fmt.Println(TAG2,userNum,"WsSessionHandler msg['command']="+msg["command"],otherCws)
		switch msg["command"] {
		case "connect":
			// create unique clientId
			myClientId = generateId()
			// send "ready" with unique clientId
			fmt.Println(TAG2, userNum, "WsSessionHandler connect: send ready myClientId:", myClientId)
			err := websocket.Message.Send(cws, fmt.Sprintf(`{"command":"ready","clientId": "%s"}`, myClientId))
			if err != nil {
				fmt.Println(TAG2, userNum, "WsSessionHandler connect: websocket.Message.Send err:", err)
			}

		case "subscribe":
			// if a call comes in, rtcCallee.js will generate a link "incoming chat from..."
			// the callee, clicking on this link, will be forwarded to rtcchat.js
			// (see: getUrlParameter('room') and subscribeRoom())
			// from where roomName + linkType will be forwarded here
			roomName = msg["room"]
			linkType = msg["linkType"]
			fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: roomName="+roomName+" linkType="+linkType)

			r, ok2 := roomInfoMap[roomName]
			if !ok2 {
				// no entry for roomName = user 1: create new map entry (roomname -> clientid)
				// TODO: retrieve and remind serverRoutedMessaging state	- OUTDATED
				userNum = 1
				fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: new room="+roomName+" cliId=", myClientId)
				var r roomInfo
				r.clientId = myClientId
				r.linkType = linkType
				r.cws = cws
				r.users = 1
				abortCmdChan = make(chan bool)

				// send roomName to all adminChannelsMap
				for _, value := range adminChannelsMap {
					value <- roomName
				}

				fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: send roomclients")
				err1 := websocket.Message.Send(cws,
					fmt.Sprintf(`{"command":"roomclients", "room":"%s", "clients":[]}`, roomName))
				if err1 != nil {
					fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: websocket.Message.Send", err1)
				} 
				/* ringtoneFile only needed for local (at home) deployment
				else {
					if ringtoneFile != "" {
						fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: play ringtoneFile", ringtoneFile)
						newRoomCmd = exec.Command("play", ringtoneFile, "repeat", "8")
						go waitForCmd(&newRoomCmd, abortCmdChan)
					}
				}*/

				// store abortCmdChan and newRoomCmd in roomInfo, so WsSessionHandler of user 2 can stop ringing
				r.abortCmdChan = abortCmdChan
				r.newRoomCmd = newRoomCmd
				roomInfoMap[roomName] = r

			} else {
				// found roomInfoMap[roomName]: this is the 2nd user to subscribe
				// send to same client: "roomclients" with array of clients in room
				userNum = 2
				fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: existroom="+roomName+" cliId="+myClientId)
				if otherCws==nil {
					otherCws = r.cws
					fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: otherCws=",otherCws)
					r.cws = cws
				}
				newRoomCmd = r.newRoomCmd
				abortCmdChan = r.abortCmdChan
				r.users = 2			
				if(linkType!=r.linkType) {
					// if both clients disagree on the linkType, make both use relayed
					fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: linkType mismatch 2:"+linkType+" 1:"+r.linkType)
					linkType="relayed"
					r.linkType="relayed"
				} else {
					fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: linkType same 2:"+linkType+" 1:"+r.linkType)
				}
				roomInfoMap[roomName] = r

				// TODO: send other clients serverRoutedMessaging state		-	OUTDATED

				// send to this client: roomclients array
				clientArray := fmt.Sprintf(`[{"clientId": "%s"}]`, r.clientId)
				json := fmt.Sprintf(`{"command":"roomclients", "room":"%s", "clients":%s}`, roomName, clientArray)
				fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: send this json="+json)
				err1 := websocket.Message.Send(cws,json)
				// TMTMTM this arrives at cws, but it is the last ws received by that client
				if err1 != nil {
					fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: websocket.Message.Send", err1)
					continue
				}

				// send to other client in this room: "presence" with data.state ("online") + data.client
				fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: send presence online...")
				clientInfo := fmt.Sprintf(`{"clientId":"%s"}`, myClientId)
				json2 := fmt.Sprintf(`{"command":"presence", "state": "online", "client": %s}`, clientInfo)
				fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: send other json="+json2)
				err2 := websocket.Message.Send(otherCws,json2)
				// TMTMTM this arrives at otherCws
				if err2 != nil {
					fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: websocket.Message.Send", err2)
					continue
				}
				fmt.Println(TAG2,userNum, "WsSessionHandler subscribe: cws=",cws)

				/* newRoomCmd only needed for local (at home) deployment
				if newRoomCmd != nil {
					// stop newRoomCmd -> stop ringing
					fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: stop ringing", myClientId)
					abortCmdChan <- true
				} else {
					fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: not stop ringing", myClientId)
				}
				*/

			}
			fmt.Println(TAG2, userNum, "WsSessionHandler subscribe: done otherCws:",otherCws)

		case "messageForward":
			// send "messageForward" and forward msg["data"]
			if otherCws == nil {
				// the 1st time user 1 does a messageForward, it does not yet know otherCws
				r, ok2 := roomInfoMap[roomName]
				if ok2 {
					fmt.Println(TAG2, userNum, "WsSessionHandler messageForward: set otherCws roomName="+roomName)
					otherCws = r.cws
					linkType = r.linkType  // both clients use the same linkType now
				} else {
					fmt.Println(TAG2, userNum, "WsSessionHandler messageForward: !otherCws !roomName="+roomName)
				}
			} else {
				fmt.Println(TAG2,userNum, "WsSessionHandler messageForward: otherCws!",otherCws,"myClientId:", myClientId)
			}

			fmt.Println(TAG2,userNum, "WsSessionHandler messageForward: cws=",cws)

			message, err := json.Marshal(msg["message"])
			if err != nil {
				fmt.Println(TAG2, userNum, "WsSessionHandler messageForward: json.Marshal err:", err)
				continue
			}
			messageString := fmt.Sprintf("%s", message)
			fmt.Println(TAG2,userNum, "WsSessionHandler messageForward: message:"+messageString)

			// UDP-RELAY proxy

			// this is where we can modify the content of message
			// which will allow us to replace the UDP target addr for both clients
			// we would start one UDP-forwarder-service per client
			// c,laddr,err = UdpProxy()
			// - have it forward all incoming data to the original clients addr:port
			// go UdpProxyWorker(c, cliaddr)
			// - and patch the forwarder's service-addr:port (laddr) into the message
			// at some point we will need to exit both services

			// entry to patch:
			// line starts with "a=candidate:" and contains "typ srflx"
			// we patch the "addr port" right before " typ srflx"
			// a=candidate:1 1 UDP 1692467199 92.201.118.190 35043 typ srflx raddr 192.168.3.224 rport 35043\\r\\n
			// a=candidate:1 2 UDP 1692467198 92.201.118.190 53857 typ srflx raddr 192.168.3.224 rport 53857\\r\\n


			// the requested linkType is sent from rtccallee.js: case "newRoom":...
			// via rtcchat.js: linkType=getUrlParameter('typ') and subscribeRoom() socket.send()
			if linkType == "p2p" {
				fmt.Println(TAG2, userNum, "WsSessionHandler messageForward: ##### P2P mode ###### ")
				if(userNum<2) {
					sendConsoleMessage(cws, otherCws, "using p2p rtc link...")
				}

			} else {
				// UDP-RELAY mode: start one UDP proxy per "typ srflx" candidate
				if(userNum<2) {
					sendConsoleMessage(cws, otherCws, "using relayed rtc link...")
				}
				hostAddrIP4 := HostAddrIP4("") // from stun.go
				var hostAddr = fmt.Sprintf("%d.%d.%d.%d",
					hostAddrIP4[0], hostAddrIP4[1], hostAddrIP4[2], hostAddrIP4[3])
				fmt.Println(TAG2, userNum, "WsSessionHandler messageForward: ##### PATCH SDP's ###### hostAddr=", hostAddr)
				var addr string
				idx := strings.Index(messageString, "typ srflx")
				if idx < 0 {
					//fmt.Println(TAG2, "WsSessionHandler messageForward: NOT FOUND 'typ srflx'")
					if(userNum<2) {
						sendConsoleMessage(cws, otherCws, "no srflx entries - using p2p link")
					}
				} else {
					substr := messageString[0:idx]
					//fmt.Println(TAG2,"substr="+substr)
					f := strings.Split(substr, " ")
					elements := len(f)
					//fmt.Println(TAG2,"elements=",elements)
					if elements <= 2 {
						fmt.Println(TAG2, "WsSessionHandler messageForward: NOT FOUND enough elements=", elements)
						if(userNum<2) {
							sendConsoleMessage(cws, otherCws, "linktype relayed failed: SDP element count")
						}
					} else {
						// addr = the host of the 1st "typ srflx" entry
						addr = f[elements-3]
						fmt.Println(TAG2, "addr found in SDP "+addr)

						var localaddr string = ""
						idx2 := strings.Index(messageString, "typ host")
						if idx2 >= 0 {
							substr2 := messageString[0:idx2]
							//fmt.Println(TAG2,"substr2="+substr2)
							f2 := strings.Split(substr2, " ")
							elements2 := len(f2)
							//fmt.Println(TAG2,"elements2=",elements2)
							if elements2 > 0 {
								// localaddr = the 192.168.x.x addr of the 1st "typ host" entry
								localaddr = f2[elements2-3]
								fmt.Println(TAG2, "localaddr="+localaddr)
							}
						}

						// we replace all addr entries (usually 3) to point to our proxy
						portArray := make([]string, 16)
						proxyCount := 0
						f = strings.Split(messageString, " ")
						var count = 0
						for idx, word := range f {
							if strings.HasPrefix(word, addr) {
								if count == 0 {
									f[idx] = hostAddr + f[idx][len(addr):]
								} else if count == 1 {
									f[idx] = hostAddr
									portArray[proxyCount] = f[idx+1] // port
									//f[idx+1] = fmt.Sprintf("%d",udpAddr.Port)
									proxyCount++
								} else {
									f[idx] = hostAddr
									portArray[proxyCount] = f[idx+1] // port
									//f[idx+1] = fmt.Sprintf("%d",udpAddr.Port)
									proxyCount++
								}
								count++
								//fmt.Println(TAG2,"replaced addr at idx=",idx,f[idx],f[idx+1])

							} else if localaddr != "" && word == localaddr {
								// we also replace localaddr to enforce relayed mode also for
								// two clients in the same network
								f[idx] = "192.168.251.251"
							}
						}

						// now we start 1 UDP proxy for each replaced "typ srflx" entry
						if proxyCount > 0 {
							// start UDP proxies
							fmt.Println(TAG2, "start udp proxies", proxyCount)
							for i := 0; i < proxyCount; i++ {
								_, err := UdpProxy(hostAddr, addr, portArray[i], sessionNumber, i)
								if err != nil {
									fmt.Println(TAG2, "UdpProxy err", err)
									// TODO:
								}
							}

							newMessageString := ""
							for _, word := range f {
								newMessageString += word + " "
							}
							messageString = newMessageString

						} else {
							if(userNum<2) {
								sendConsoleMessage(cws, otherCws, "linktype relayed failed: addr not found")
							}
						}

						sessionNumber++
						if sessionNumber > 50000 {
							sessionNumber = 0
						}

					}
				}
			}

			msgType := msg["msgType"]
			json := fmt.Sprintf(`{"command":"messageForward", "msgType":"%s", "message": %s}`,msgType,messageString)
			fmt.Println(TAG2,userNum,"WsSessionHandler messageForward json:",json)
			err2 := websocket.Message.Send(otherCws, json)
			if err2 != nil {
				fmt.Println(TAG2, userNum,"WsSessionHandler messageForward: websocket.Message.Send err:", err2)
			} else {
				fmt.Println(TAG2, userNum,"WsSessionHandler messageForward: sent done")
			}
			fmt.Println(TAG2, userNum,"WsSessionHandler messageForward: done")

		case "stopRing":
			fmt.Println(TAG2, userNum,"WsSessionHandler stopRing msg=", msg)
			calleekey := msg["calleekey"]
			fmt.Println(TAG2, userNum,"WsSessionHandler stopRing calleekey=", calleekey)
			if calleekey != "" {
				calleeCws, ok := CalleeMap[calleekey]
				if ok {
					fmt.Println(TAG2, userNum,"WsSessionHandler stopRing websocket.Message.Send()")
					websocket.Message.Send(calleeCws, `{"command":"stopRing"}`)
				}
			}

		case "forRing":
			fmt.Println(TAG2, userNum,"WsSessionHandler forRing msg=", msg)
			calleekey := msg["calleekey"]
			fmt.Println(TAG2, userNum,"WsSessionHandler forRing calleekey=", calleekey)
			if calleekey != "" {
			    var ok = false
				forRingCws, ok = CalleeMap[calleekey]
				if !ok {
				    forRingCws = nil
				}
            }
			if forRingCws==nil {
    			fmt.Println(TAG2, userNum,"WsSessionHandler forRing no forRingCws")
			} else {
    			fmt.Println(TAG2, userNum,"WsSessionHandler forRing forRingCws set")
			}
		}
	}

	// send stop ringing in case caller has just disappeared
	if forRingCws!=nil {
		// this will be handled in rtccallee.js
		fmt.Println(TAG2, "WsSessionHandler end of session stopRing")
	    websocket.Message.Send(forRingCws, `{"command":"stopRing"}`)
	} else {
		fmt.Println(TAG2, "WsSessionHandler end of session stopRing no forRingCws")
	}

	if roomName != "" {
		// the last user leaving the room must clean up
		r := roomInfoMap[roomName]
		if r.users > 0 {
			r.users--
		}
		if r.users > 0 {
			roomInfoMap[roomName] = r
		} else {
			fmt.Println(TAG2, "WsSessionHandler delete room", roomName)
			delete(roomInfoMap, roomName)

			// send empthy roomName to all adminChannelsMap
			for _, value := range adminChannelsMap {
				value <- ""
			}
		}
	}
	if newRoomCmd != nil {
		fmt.Println(TAG2, "WsSessionHandler done myClientId set abortCmdChan", myClientId)
		// stop newRoomCmd -> stop ringing
		abortCmdChan <- true
	} else {
		fmt.Println(TAG2, "WsSessionHandler done myClientId !set abortCmdChan", myClientId)
	}
	done <- true
}

func sendConsoleMessage(cws *websocket.Conn, otherCws *websocket.Conn, msg string) {
	json := fmt.Sprintf(`{"command":"consoleMessage", "message": "%s"}`, msg)
	fmt.Println(TAG2, "sendConsoleMessage="+msg)
	if cws != nil {
		websocket.Message.Send(cws, json)
	}
	if otherCws != nil {
		websocket.Message.Send(otherCws, json)
	}
}

/*
func waitForCmd(cmd **exec.Cmd, abortCmdChan <-chan bool) {
	// as long as the cmd is running, (*cmd) is not null
	// (*cmd) can be killed from outside by: abortCmdChan <- true
	err := (*cmd).Start()
	if err == nil {
		cmddone := make(chan error)
		go func() {
			// wait for shell command to end
			fmt.Println(TAG2, "waitForCmd cmd.Wait()...")
			cmddone <- (*cmd).Wait()
			fmt.Println(TAG2, "waitForCmd cmd.Wait() done")
			*cmd = nil
		}()
		select {
		// wait for shell command to end -or- for client to leave room (abortCmdChan)
		case err := <-cmddone:
			// shell command exited normally
			fmt.Println(TAG2, "waitForCmd cmddone err=", err)
		case <-abortCmdChan:
			// client has left early: kill shell command
			fmt.Println(TAG2, "waitForCmd Process.Kill()...")
			if err := (*cmd).Process.Signal(syscall.SIGKILL); err != nil {
				fmt.Println(TAG2, "waitForCmd failed to kill", err)
			}
			fmt.Println(TAG2, "waitForCmd wait cmddone...")
			<-cmddone // allow goroutine to exit
			fmt.Println(TAG2, "waitForCmd wait cmddone done")
		}
	} else {
		fmt.Println(TAG2, "waitForCmd cmd.Start() failed", err)
	}
}
*/
