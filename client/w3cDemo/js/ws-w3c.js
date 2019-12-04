available_signals = {
    "Speed":"#ws-speedometer"
};

class W3CWebSocket extends W3CClient{
    constructor(debug){
        super(debug);
        
     }

    setup(){
        this.socket = new WebSocket('ws://192.168.150.131:8080/');
        this.socket.onclose = this.websocket_onclose;
        this.socket.onmessage = this.websocket_onmessage;

    }

    websocket_onclose(event){
        console.error('Socket closed');
    }
    websocket_onmessage(event){
        var request = JSON.parse(event.data);
        W3CClient.prototype.handleRequest(request) // does nothing atm
        W3CWebSocket.prototype.addLog(event.data, 'ws-receive-log-list');
        switch(request.action){
			
            case "get":
                if (request.path in available_signals) {
                    W3CWebSocket.prototype.updateSpeedometerValue(self.available_signals[request.path], request.value);
                }
                break;
            case "subscription":
				
                if (request.path in available_signals) {
                    W3CWebSocket.prototype.updateSpeedometerValue(self.available_signals[request.path], request.value);
                }
                break;
            default:
        }
    }

    requestGet(path){
		//{"action":"get", "path":"Vehicle.Cabin.Door.*.*.IsOpen", "requestId":"123"}
		var msg = JSON.stringify({ 'action': 'get', 'path': path, 'requestId': 234});
		this.socket.send(msg);
		this.addLog(msg, 'ws-send-log-list');
	}

	requestSet(path, value){
		//{"action":"set", "path":"Vehicle.Cabin.Door.Row1.Right.IsOpen", "value":"999", "requestId":"234"}
		socket.send(JSON.stringify({ 'action': 'set', 'path': path, 'value': value, 'requestId': 234}));
	}

	subscribe(path){
		//{"action":"subscribe", "path":"Vehicle.Cabin.Door.Row1.Right.IsOpen", "requestId":"234"}
		var msg = JSON.stringify({ 'action': 'subscribe', 'path': path, 'requestId': 234})
		socket.send(msg);
		this.addLog(msg, 'ws-send-log-list');
	}

	unsubscribe(subscriptionid){
		//{"action":"unsubscribe", "subscriptionId":"789", "requestId":"234"}
		socket.send(JSON.stringify({ 'action': 'unsubscribe', 'subscriptionid': subscriptionid, 'requestId': 234}));
	}
	addLog(value, log_id){
		console.log(value);
        var log_item = document.createElement('li');
        log_item.appendChild(document.createTextNode(value));
        log_item.classList.add("list-group-item");
        
        var sendLog = document.getElementById(log_id);
        sendLog.append(log_item);
        sendLog.scrollTop = sendLog.scrollHeight;
        //$(log_id).append(log_item);		
	}
	
	
    updateSpeedometerValue(id, value){
		$(id).val(value);
		$(id).trigger("change");
	}
}
