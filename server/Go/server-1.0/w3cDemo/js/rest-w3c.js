available_signals = {
    "Speed":"#rest-speedometer"
};

class W3CRestClient extends W3CClient{
    constructor(debug){
        super(debug);

     }
    setup(){
        //this.send_post_request("Vehicle/Cabin/Door/Row1/Right/IsOpen");
    }

    send_get_request(request){
        $.ajax({
            type: "GET",
            url: "http://localhost:8888/" + request,
            dataType: 'jsonp',
            accept: 'application/json',
            success: this.message_success,
            error: function()
            {
                console.log("error");
            },
        
        });
    }
    
    send_post_request(request, value){
        $.ajax({
            type: "POST",
            url: "http://localhost:8888/" + request,
            dataType: 'jsonp',
            data: {"value": value},
            accept: 'application/json',
            success: this.message_success,
            error: function()
            {
                console.log("error");
            },
        
        });
    }

    message_success(request){
        console.log(request);
        //var request = JSON.parse(event);
        W3CClient.prototype.handleRequest(request) // does nothing atm

        switch(request.action){
            case "get":
                if (request.path in available_signals) {
                    W3CRestClient.prototype.updateSpeedometerValue(self.available_signals[request.path], request.value);
                }
                break;
            case "subscription":
                if (request.path in available_signals) {
                    W3CRestClient.prototype.updateSpeedometerValue(self.available_signals[request.path], request.value);
                }
                break;
            default:
        }
    }
	
	requestGet(path){
		//{"action":"get", "path":"Vehicle.Cabin.Door.*.*.IsOpen", "requestId":"123"}
		send_get_request(path);
	}

	requestSet(path, value){
		//{"action":"set", "path":"Vehicle.Cabin.Door.Row1.Right.IsOpen", "value":"999", "requestId":"234"}
		send_post_request(path, value);
	}

	subscribe(path){
		//{"action":"subscribe", "path":"Vehicle.Cabin.Door.Row1.Right.IsOpen", "requestId":"234"}
		//socket.send(JSON.stringify({ 'action': 'subscribe', 'path': path, 'requestId': 234}));
	}

	unsubscribe(){
		//{"action":"unsubscribe", "subscriptionId":"789", "requestId":"234"}
		//socket.send(JSON.stringify({ 'action': 'unsubscribe', 'path': path, 'requestId': 234}));
	}

    updateSpeedometerValue(id, value){
		$(id).val(value);
		$(id).trigger("change");
	}
}
