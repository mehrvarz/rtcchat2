// rtcchat2 caller-enter-name.js
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
    console.log("start: host",host);
    //console.log("start: wsCallerPort:",wsCallerPort," key:",key);

    calleeURL = window.location.href+"callee:"+calleeKey;
    callerURL = calleeURL.split(':'+wsCalleePort)[0]+":"+wsCallerPort+"/call:"+callerKey;

    console.log("onLoad() calleeURL="+calleeURL+" callerURL="+callerURL+" keyKey="+keyKey);

	var calleeLabel = "<div id='calleeId' style='text-align:left'>"+calleeURL+"</div>";
    $('#calleeId').replaceWith(calleeLabel)
	var callerLabel = "<div id='callerId' style='text-align:left'>"+callerURL+"</div>";
    $('#callerId').replaceWith(callerLabel)

	connectToCalleeService();
});

function copyCallee() {
	window.prompt("This is your callee-URL. You can use this URL to receive chat calls.\n\nCopy to clipboard: Ctrl+C, Enter", calleeURL);
}

function copyCaller() {
	window.prompt("Your caller URL is like a phone number. Share it with friends.\n\nCopy to clipboard: Ctrl+C, Enter", callerURL);
}

function connectToCalleeService() {
	// try to connect to callee service
    var hostAddr = host+":"+wsCalleePort;
    var	socketServerAddress;
	if(window.location.href.indexOf("https://")==0)
		socketServerAddress = "wss://"+hostAddr+"/ws";
	else
		socketServerAddress = "ws://"+hostAddr+"/ws";
    console.log("connecting to callee service",hostAddr);
    //writeToChatLog("connecting to callee service "+hostAddr+" ...", "text-success");
    socket = new WebSocket(socketServerAddress);
    if(!socket) {
	    console.log("failed to connect to callee service",hostAddr);
		window.setTimeout(function(){
			connectToCalleeService();
		},2000);
	}

	socket.onopen = function () {
	    console.log("connected to callee service",hostAddr);
	    //writeToChatLog("connected to callee service "+hostAddr, "text-success");

		lastServerAction = new Date().getTime();
	    // start heartbeat (send "alive?" requests, if last "connect" is older than)
	    checkHeartBeats();
	};

	socket.onerror = function () {
	    console.log("failed to connect to callee service",hostAddr);
        //writeToChatLog("failed to create websocket connection", "text-success");
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
			    alert("Your keys have been generated!\n\n"+
			    	  "You will now be transfered to the callee service.\n"+
			    	  "As long as you keep it running, others will be able to call you.\n"+
			    	  "Please make sure you bookmark the following page.\n"+
			    	  "Don't share your callee URL with anyone.\n");
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
			// must reconnect
		    console.log("disconnected from admin server");
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
	// caller has entered her name
    makeCall()
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
    // wait for response - see: case "activateConfirm":
    // TODO: can socket.send() fail?
}

