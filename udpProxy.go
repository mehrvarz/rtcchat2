// rtcchat2 udpProxy.go
// Copyright 2013 Timur Mehrvarz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rtcchat2

import (
	"fmt"
	"net"
	"strings"
	"time"
)

var TAG5 = "UdpProxy"

var udpConns = make(map[string]*net.UDPConn, 1000)
var runningThreads = 0  // for debugging only

func UdpProxy(hostAddr string, addr string, port string, sessionNumber int, proxyNumber int) (*net.UDPAddr, error) {
    srcaddr := hostAddr+":"+port
    target := addr+":"+port
    proxyname := fmt.Sprintf("T%d/%d", sessionNumber, proxyNumber)

	// start listen on some UDP port
	laddr, err := net.ResolveUDPAddr("udp4", srcaddr)
	if err != nil {
		return nil, err
	}
	fmt.Println(TAG5, proxyname, "from=", laddr, "to=", target)

	c, er2 := net.ListenUDP("udp", laddr)
	if er2 != nil {
		return laddr, er2
	}
	//fmt.Println(TAG5,proxyname,"laddr",laddr)

	go UdpProxyWorker(c, target, sessionNumber, proxyNumber, proxyname)
	return laddr, nil
}

func UdpProxyWorker(c *net.UDPConn, target string, sessionNumber int, proxyNumber int, proxyname string) {
	// start listen on some UDP port
	targetaddr, err := net.ResolveUDPAddr("udp4", target)
	if err != nil {
		fmt.Println(TAG5, proxyname, "UdpProxyWorker ResolveUDPAddr err", err)
		return
	}
	runningThreads++
	fmt.Println(TAG5, proxyname, "UdpProxyWorker targetaddr runningThreads", targetaddr, runningThreads)

    udpConnName := fmt.Sprintf("%d-%d-%d", sessionNumber,proxyNumber,0)
	udpConns[udpConnName] = c
	fmt.Println(TAG5, proxyname, "########## STORED udpConns[]",udpConnName)

    // enable read timeout only for first read
	c.SetReadDeadline(time.Now().Add(60 * time.Second))

	var clientSourceAddr *net.UDPAddr = nil
	var c2 *net.UDPConn = nil
	var subStarted = false
	var retError error = nil

	for retError == nil && c != nil {
		//fmt.Println(TAG5,proxyname,"Read...",c.LocalAddr().String())
		var buf [16240]byte
		l, srcaddr, erd := c.ReadFromUDP(buf[0:])
		if erd != nil {		
			fmt.Println(TAG5, proxyname, "c.ReadFromUDP err", erd)
			retError = erd

			// this means one client has gone off
			// TODO: only if erd.Error() contains "connection refused" - not on "address already in use"
			// TODO: tear down the partner threads of this session now
			i := 0
            udpConnName = fmt.Sprintf("%d-%d-%d", sessionNumber,i,0)
            _,ok := udpConns[udpConnName]
            for ok {
        		fmt.Println(TAG5, proxyname, "########## FOUND udpConns[]",udpConnName)
        		udpConns[udpConnName].Close()
        		udpConns[udpConnName] = nil
                udpConnName = fmt.Sprintf("%d-%d-%d", sessionNumber,i,1)
                _,ok = udpConns[udpConnName]
                if(ok) {
            		fmt.Println(TAG5, proxyname, "########## FOUND udpConns[]",udpConnName)
            		udpConns[udpConnName].Close()
            		udpConns[udpConnName] = nil
                }

        		i++
                udpConnName = fmt.Sprintf("%d-%d-%d", sessionNumber,i,0)
                _,ok = udpConns[udpConnName]
        	}
			break
		}
		
		//fmt.Println(TAG5,proxyname,"write len",idx,"\n",hex.Dump(buf[0:l]))  // import "encoding/hex"

		if l > 0 {
			fmt.Println(TAG5, proxyname, "c.ReadFromUDP srcaddr len", srcaddr, l) //,buf[0:l])

            // extend timeout very generously from now on
        	c.SetReadDeadline(time.Now().Add(60*60*3 * time.Second))    // 3 hours

			if srcaddr.Port != targetaddr.Port {
				// if we receive data NOT from our target (this is our client), we forward it to our target
				if clientSourceAddr == nil {
					// the 1st data is coming from our client: keep the clients address
					clientSourceAddr = srcaddr
					fmt.Println(TAG5, proxyname, "DialUDP", targetaddr)
					var erx error
					dialaddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", srcaddr.Port))
					if err != nil {
						fmt.Println(TAG5, proxyname, "dialaddr FAIL ################# err", err)
					}
					c2, erx = net.DialUDP("udp", dialaddr, targetaddr)
					if erx != nil {
						fmt.Println(TAG5, proxyname, "DialUDP err", erx)
						// TODO: happens frequently: "udp 46.115.54.228:59971: address already in use"
						if strings.Index(erx.Error(),"address already in use")>=0 {
						    // this is expected
						}
						c.Close()
						c = nil
						retError = erx
						break
					}

                    udpConnName := fmt.Sprintf("%d-%d-1", sessionNumber,proxyNumber)
		            _,ok := udpConns[udpConnName]
		            if(!ok) {
                		udpConns[udpConnName] = c2
                		fmt.Println(TAG5, proxyname, "########## STORED udpConns[]",udpConnName)
                	}

					fmt.Println(TAG5, proxyname, "DialUDP c2.LocalAddr c2.RemoteAddr", 
									c2.LocalAddr(), c2.RemoteAddr())

				} /*else if(srcaddr!=clientSourceAddr && srcaddr!=targetaddr) {
				    // this is for safety, so that our UDP connection cannot be deranged by some 3rd parties
				    // TODO: if is true, even if srcaddr==clientSourceAddr
					fmt.Println(TAG5,proxyname,"UNKNOWN UDP CLIENT - ignore",srcaddr,clientSourceAddr,targetaddr)
					continue
				}*/

				if c2 != nil {
					wlen, ere := c2.Write(buf[0:l])
					if ere != nil {
						fmt.Println(TAG5, proxyname, "c2.Write targetaddr err", ere)
						c2.Close()
						c2 = nil
						//c.Close()
						//c=nil
						//retError = ere
						break
					}
					fmt.Println(TAG5, proxyname, "c2.Write done targetaddr len", targetaddr, wlen)

					if !subStarted {
						subStarted = true
						// a thread for c2.ReadFromUDP() -> c.Write()
						go func() {
							runningThreads++
							fmt.Println(TAG5, proxyname, "sub start thread c2 -> c runningThreads=", runningThreads)
							for c != nil && c2 != nil {
								var buf [16240]byte
								l, srcaddr2, erd := c2.ReadFromUDP(buf[0:])
								if erd != nil {
									fmt.Println(TAG5, proxyname, "sub ###### c2.ReadFromUDP err",
										c2.LocalAddr(), c2.RemoteAddr(), erd)
									// "read udp 109.74.203.226:52414: connection refused"
									// but should be reading from targetaddr=92.201.118.190
									c2.Close()
									c2 = nil
									c.Close()
									break
								} else {
									fmt.Println(TAG5, proxyname, "sub c2.ReadFromUDP done from srcaddr2 l", srcaddr2, l)
									//if(srcaddr.Port!=targetaddr.Port) {
									if l > 0 {
										wlen, ere := c.WriteTo(buf[0:l], srcaddr)
										if ere != nil {
											fmt.Println(TAG5, proxyname, "sub c.WriteTo srcaddr error", ere)
											c2.Close()
											c2 = nil
											c.Close()
											//c=nil
											//retError = ere
											break
										}
										fmt.Println(TAG5, proxyname, "sub c.WriteTo done srcaddr len", srcaddr, wlen)
									}
								}
							}
							runningThreads--
							fmt.Println(TAG5, proxyname, "sub end thread runningThreads=", runningThreads)
						}()
					}
				}
			} /*else {
				// otherwise (we receive data from our target), we forward it to our client
				wlen, ere := c.WriteTo(buf[0:l], clientSourceAddr)
				if ere != nil {
					fmt.Println(TAG5,proxyname,"write to clientSourceAddr error", ere)
					c.Close()
					retError = ere
					break
				}
				fmt.Println(TAG5,proxyname,"written to clientSourceAddr len",wlen)
			}*/
		}
	}

	if c != nil {
		fmt.Println(TAG5, proxyname, "exit UdpProxyWorker c.Close()")
		c.Close()
		c = nil
	}
	if c2 != nil {
		fmt.Println(TAG5, proxyname, "exit UdpProxyWorker c2.Close()")
		c2.Close()
		c2 = nil
	}

	runningThreads--
	fmt.Println(TAG5, proxyname, "exit UdpProxyWorker done runningThreads=", runningThreads)
}
