rtc chat v2
===========

A WebRTC chat service written in Go

rtc chat establishes end-to-end encrypted, P2P and relayed communication links.

More info: [http://mehrvarz.github.io/rtcchat2](http://mehrvarz.github.io/rtcchat2/)

Install
-------

go get github.com/mehrvarz/rtcchat2

go get github.com/steveyen/gkvlite

cd $GOPATH/src/mehrvarz/rtcchat2

go run rtcchat/gkvCreate.go

./run

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

