// rtcadmin.js
// Copyright 2013 Timur Mehrvarz <timur.mehrvarz@riseup.net>

var host;
var wsPort = {{.SigPort}}; 		   // default=8077, will be patched by rtcSignaling.go service
var wsAdminPort = {{.SigPort}} +1; // default=8078, will be patched by rtcSignaling.go service
var secureRedirect = {{.SecureRedirect}}
var socket = null;
var lastServerAction = 0;

$(function(){
	// init
	host = location.hostname;
    console.log("start: host="+host+" wsAdminPort="+wsAdminPort);
	connectToAdminServer();
});

function connectToAdminServer() {
	// try to connect to admin server
    var hostAddr = host+":"+wsAdminPort;
    var	socketServerAddress;
	if(window.location.href.indexOf("https://")==0)
		socketServerAddress = "wss://"+hostAddr+"/ws";
	else
		socketServerAddress = "ws://"+hostAddr+"/ws";
    console.log("connecting to admin server",hostAddr);
    writeToChatLog("connecting to admin server "+hostAddr+" ...", "text-success");
    socket = new WebSocket(socketServerAddress);
    if(!socket) {
	    console.log("failed to connect to admin server",hostAddr);
		window.setTimeout(function(){
			connectToAdminServer();
		},2000);
	}

	socket.onopen = function () {
	    console.log("connected to admin server",hostAddr);
	    writeToChatLog("connected to admin server "+hostAddr, "text-success");

		lastServerAction = new Date().getTime();
	    // start heartbeat (send "alive?" requests, if last "connect" is older than)
	    checkHeartBeats();
	    
	    // announce our availability for incoming calls
	    // key (public caller key) taken from index.html, patched by calleeService.go
	    console.log("announce ",key);
	    socket.send(JSON.stringify({command:'announce', uniqueID: key}));
		// this will be processed in calleeService: case "announce":	    
	};

	socket.onerror = function () {
	    console.log("failed to connect to admin server",hostAddr);
        writeToChatLog("failed to create websocket connection", "text-success");
		window.setTimeout(function(){
			connectToAdminServer();
		},3000);
	}
    socket.onmessage = function(m) { 
        var data = JSON.parse(m.data);
    	//console.log("socket message raw:", data);
    	
    	switch(data.command) {
		case "alive!":
			// this is the host confirming connect or alive
			//console.log("connect:");
			// reset heartbeat timeout
			lastServerAction = new Date().getTime();
			break;

		case "info":
			var msg = data.msg;
			console.log("info=",msg);
	        writeToChatLog(msg, "text-success");
	        break;

		case "newRoom":
			// someone is calling us
			// this event started in caller-enter-name.js: makeCall()
			// and is being delivered via callerService.go: case "call":
			var callerName = data.callerName;
			var roomName = data.roomName;
			var linkType = data.linkType;
			console.log("newRoom: roomName="+roomName+" callerName="+callerName+" linkType="+linkType);
			lastServerAction = new Date().getTime();
			if(roomName!="" && callerName!="") {
				var prot = "http";
				if(secureRedirect) 
				    prot = "https";

				// answer incoming call action
				// the following links will be processed in /html/callee/rtccallee.js
            	// callee, clicking on these links, will be forwarded to rtcchat.js (see: getUrlParameter('room'))
            	// from where roomName + linkType will be forwarded to rtcSignaling.go (see: case "subscribe":)

            	// answer in relayed mode
				var msg = "<a href=\""+prot+"://"+location.hostname+":"+wsPort+
								"/?room="+roomName+"&amp;key="+key+"&amp;linktype="+linkType+"\" "+
								"target=\"_blank\">incoming chat from "+callerName+"</a>";

            	if(linkType=="p2p") {
	            	// answer in p2p mode
					msg += " - <a href=\""+prot+"://"+location.hostname+":"+wsPort+
								"/?room="+roomName+"&amp;key="+key+"&amp;linktype=relayed\" "+
								"target=\"_blank\">relayed</a>";
				}

				// offer callee to just mute the ringing
				msg += " - <a href=\"javascript:stopRing()\">mute</a>";

            	writeToChatLog(msg, "text-success");

            	// start ringing in an endless loop... (see "stopRing" and stopRing() below for more info)
           	    document.getElementById('audiotag').play();

            } else {
            	writeToChatLog("room closed", "text-success");
            }
	        break;

		case "stopRing":
			// this is how rtcchat.js stops us ringing, when a call is answered:
			// the above link(s) contains a parameter 'key='
			// when the callee clicks on this link to answer an incoming call,
			// rtcchat.js will discover the key via getUrlParameter('key')
			// it will forward the key to the rtcSignaling.go service via socket.send({command:'stopRing'...})
			// rtcSignaling.go will do: calleeCws = rtcredirect.CalleeMap[calleekey]
			// as well as: websocket.Message.Send(calleeCws, `{"command":"stopRing"}`)
			// which will end up below at "stopRing"
			console.log("stopRing");
       	    document.getElementById('audiotag').pause();
           	writeToChatLog("chat call was handled", "text-success");
	        break;
		}
    }
}

function stopRing() {
	// this is how a callee just mutes the ringing
	console.log("stopRing() socket="+socket);
	document.getElementById('audiotag').pause();
   	writeToChatLog("chat call was muted", "text-success");
}

function checkHeartBeats() {
	window.setTimeout(function(){
		var timeSinceLastServerAction = new Date().getTime() - lastServerAction;
		if(timeSinceLastServerAction>6000) {
			// must reconnect
		    console.log("disconnected from admin server");
			connectToAdminServer();
			return;
		}
		
		if(timeSinceLastServerAction>3000) {
		    console.log("check if admin server still alive...");
	    	socket.send(JSON.stringify({command:'alive?'}));
		}
		checkHeartBeats();
    },500);
}

function getUrlParameter(name) {
    name = name.replace(/[\[]/,"\\\[").replace(/[\]]/,"\\\]");
    var regexS = "[\\?&]"+name+"=([^&#]*)";
    var regex = new RegExp(regexS);
    var results = regex.exec(window.location.href);
    if(results != null)
        return results[1];
    return "";
}

function linkify(text) {
    // http://stackoverflow.com/questions/37684/how-to-replace-plain-urls-with-links
    // http://benalman.com/code/test/js-linkify/
    var exp = /(\b(https?|ftp|file):\/\/[-A-Z0-9+&@#\/%?=~_|!:,.;]*[-A-Z0-9+&@#\/%=~_|])/ig;
    return text.replace(exp,"<a href='$1'>$1</a>"); 
}

function receiveMessage(msg) {
    msg = linkify(msg);
    document.getElementById('audiotag').play();
    writeToChatLog(msg, "text-info");
}

function sendMessage(msg) {
    console.log("sendMessage", msg);
    if (msg) {
        $('#messageTextBox').val("");

	    // fileReceiver
    	//var channel = new RTCMultiSession();
        //channel.send({message: msg});

        if(serverRoutedMessaging) {
        	socket.send(JSON.stringify({
        		command:'messageForward', 
			    msgType:'message', 
        		message: JSON.stringify(msg)
        	}));
            msg = linkify(msg);
            writeToChatLog(msg, "text-success");
        } else {
            if(webrtcDataChannel) {
                webrtcDataChannel.send(msg);
                msg = linkify(msg);
                writeToChatLog(msg, "text-success");
            } else {
                writeToChatLog("sendMessage failed no webrtcDataChannel", "text-success");
            }
        }
    }

    return false;
};

function sendMessageFromForm() {
    //console.log("sendMessageFromForm() -> sendMessage()",$('#messageTextBox').val());
    sendMessage($('#messageTextBox').val());
}

$('#sendMessageBtn').click(function() {
    sendMessageFromForm();
});

function getTimestamp() {
    var totalSec = new Date().getTime() / 1000;
    var hours = parseInt(totalSec / 3600) % 24;
    var minutes = parseInt(totalSec / 60) % 60;
    var seconds = parseInt(totalSec % 60);
    return result = (hours < 10 ? "0" + hours : hours) + ":" +
                    (minutes < 10 ? "0" + minutes : minutes) + ":" +
                    (seconds  < 10 ? "0" + seconds : seconds);
}

function writeToChatLog(message, message_type) {
    var msg = message;
    if(message_type!="text-success")
        msg = "other: "+message;
    document.getElementById('chatlog').innerHTML 
    	+= '<p class=\"'+message_type+'\">'+'['+getTimestamp()+'] '+msg+'</p>';
    // Scroll chat text area to the bottom on new input.
    $('#chatlog').scrollTop($('#chatlog')[0].scrollHeight);
}

