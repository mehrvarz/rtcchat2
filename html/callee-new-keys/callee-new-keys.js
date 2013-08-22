// rtcchat2 callee-new-keys.js
// Copyright 2013 Timur Mehrvarz <timur.mehrvarz@riseup.net>
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

var host;
var socket = null;
var lastServerAction = 0;
var calleeURL;
var callerURL;

$(function(){
	host = location.hostname;
    console.log("onLoad: host",host);
    calleeURL = window.location.href+"callee:"+calleeKey;
    callerURL = calleeURL.split(':'+wsCalleePort)[0]+":"+wsCallerPort+"/call:"+callerKey;
    console.log("onLoad: calleeURL="+calleeURL+" callerURL="+callerURL+" keyKey="+keyKey);
	connectToCalleeService();
});

function connectToCalleeService() {
    var hostAddr = host+":"+wsCalleePort;
    var	socketServerAddress;
	if(window.location.href.indexOf("https://")==0)
		socketServerAddress = "wss://"+hostAddr+"/ws";
	else
		socketServerAddress = "ws://"+hostAddr+"/ws";
    console.log("connecting to callee service",hostAddr);
    socket = new WebSocket(socketServerAddress);
    if(!socket) {
	    console.log("failed to connect to callee service",hostAddr);
		window.setTimeout(function(){
			connectToCalleeService();
		},2000);
	}

	socket.onopen = function () {
	    console.log("connected to callee service",hostAddr);
		lastServerAction = new Date().getTime();
	    // start heartbeat (send "alive?" requests, if last "connect" is older than)
	    checkHeartBeats();
	};

	socket.onerror = function () {
	    console.log("failed to connect to callee service",hostAddr);
		window.setTimeout(function(){
			connectToCalleeService();
		},3000);
	}
    socket.onmessage = function(m) { 
        var data = JSON.parse(m.data);
    	
    	switch(data.command) {
		case "alive!":
			// this is the host confirming connect or alive
			// reset heartbeat timeout
			lastServerAction = new Date().getTime();
			break;

		case "info":
			var msg = data.msg;
			console.log("info=",msg);
	        break;

		case "activateConfirm":
			// this is being sent by calleeService.go
			var success = data.success;
			console.log("activateConfirm success=",success);
			if(success) {
			    // success: forward to callee URL
				console.log("success activateConfirm: forward to callee URL");
				window.location.href = calleeURL;

			} else {
			    // error generating keys
				console.log("activateConfirm failed to activate keys");
			    alert('Failed to activate keys. Please reload page and try again.');
			}   
	        break;
		}
    }
}

function checkHeartBeats() {
	window.setTimeout(function(){
		var timeSinceLastServerAction = new Date().getTime() - lastServerAction;
		if(timeSinceLastServerAction>6000) {
		    console.log("disconnected from admin server");
			// we need to reconnect to server
			connectToCalleeService();
			return;
		}
		
		if(timeSinceLastServerAction>3000) {
		    console.log("check if admin server still alive...");
	    	socket.send(JSON.stringify({command:'alive?'}));
		}
		checkHeartBeats();
    },500);
}

$('#callBtn').click(function() {
    // user has entered a room name
    activateKeys()
});

function activateKeys() {
    console.log("activateKeys() keyKey="+keyKey);
    if(!socket) {
	    alert('sorry: failed to connect to server');
    	return;
    }
	// we request calleeService to call rtcchat.StoreNewKeys(keyKey)
	try {
	    socket.send(JSON.stringify({command:'activateKeys', key: keyKey}));
	} catch(e) {
		alert("websocket failed ",e)
	}
}

