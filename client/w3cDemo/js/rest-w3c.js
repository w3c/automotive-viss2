available_signals = {
    "Speed":"#rest-speedometer"
};

class W3CRestClient extends W3CClient{
    constructor(debug){
        super(debug);

     }
    setup(){

    }

    send_get_request(request){
        $.ajax({
            type: "GET",
            url: "http://192.168.150.131:8888/" + request,

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
            url: "http://192.168.150.131:8888/" + request,
            data: "123",
            success: this.message_success,
            error: function()
            {
                console.log("error");
            },
        
        });
    }

    message_success(request){
        W3CClient.prototype.handleRequest(request) // does nothing atm
        W3CRestClient.prototype.addLog(request, 'rest-receive-log-list');
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
        W3CRestClient.prototype.addLog(path, 'rest-send-log-list');
		this.send_get_request(path);
	}

	requestSet(path, value){
        W3CRestClient.prototype.addLog(path + ", data: " + value, 'rest-send-log-list');
		this.send_post_request(path, value);
	}

    addLog(value, log_id){
        var log_item = document.createElement('li');
        log_item.appendChild(document.createTextNode(value));
        log_item.classList.add("list-group-item");
        
        var sendLog = document.getElementById(log_id);
        sendLog.append(log_item);
        sendLog.scrollTop = sendLog.scrollHeight;
	}

    updateSpeedometerValue(id, value){
		$(id).val(value);
		$(id).trigger("change");
	}
}
