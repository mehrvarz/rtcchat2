rtc chat v2
===========

A WebRTC chat service written in Go

rtc chat establishes end-to-end encrypted, P2P and relayed communication links.


Installation
------------

You should have Go 1.1 installed. 

Install the following modules:

	go get github.com/mehrvarz/rtcchat2
	go get github.com/steveyen/gkvlite
	go get code.google.com/p/go.net/websocket

Initialize the callee service key/value flat file DB.

	cd $GOPATH/src/mehrvarz/rtcchat2
	go run rtcchat/gkvCreate.go


Run server without SSL keys (test mode)
---------------------------------------

rtc chat server should be run with SSL keys installed (see next section). 
For plain test purposes, you can run rtc chat server without SSL keys (and use http instead of https):

	go run rtcchat/main.go -secure=false


Run server with SSL keys (standard mode)
----------------------------------------

Create SSL keys for use with HTTPS:

	mkdir keys && cd keys
	openssl req -new -x509 -nodes -out cert.pem -keyout key.pem -days 100
	(answer questions)
	cd ..

Alternative: create symbolic links to your existing keys froms /etc/nginx

	mkdir keys && cd keys
	ln -s /etc/nginx/cert.pem cert.pem
	ln -s /etc/nginx/key.pem key.pem
	cd ..

Please note: the "keys" subfolder is expected to contain two files: "cert.pem" and "key.pem".

This is how your installation folder should look:

	rtcchat2

		html
			index.html
			spinner.gif
			bootstrap.min.css
			...

		rtcchat
			main.go
			gkvCreate.go
			...

		keys
			key.pem
			cert.pem

		rtcSignaling.go
		calleeService.go
		callerService.go
		calleePersistKey.go
		udpProxy.go
		stun.go
		LICENSE
		...

To run rtc chat server:

	go run rtcchat/main.go

Or simply:

	./run
	
Test your server 
----------------

1. Open the following URL in two browser tabs:

	https://127.0.0.1:8077
	
	http://127.0.0.1:8077 (for insecure test mode)

2. Enter the same 'secret word' in both browser tabs. You should see the two instances connect.


More info: [http://mehrvarz.github.io/rtcchat2](http://mehrvarz.github.io/rtcchat2/)

License
-------

This project uses code from:

bootstrap.js: Copyright 2012 Twitter, Inc; Apache License, Version 2.0.<br/>
jquery: Copyright jQuery Foundation and contributors; MIT License.<br/>
adapter.js + serverless-webrtc.js: Copyright 2013 Chris Ball <chris@printf.net>.<br/>
gkvlite: Copyright Steve Yen; MIT license.<br/>

For the rest:

Copyright (C) 2013 Timur Mehrvarz

Permission is hereby granted, free of charge, to any person obtaining a
copy of serverless-webrtc and associated documentation files (the "Software"),
to deal in the Software without restriction, including without limitation the
rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
sell copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.

