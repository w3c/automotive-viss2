class W3CClient { 
    constructor(debug){
        self.debug = debug;
    }

    handleRequest(request){
        switch(request.action){
            case "get":
                break;
            case "set":
                break;
            case "subscription":
                break;
            default:
        }

        if (self.debug){
            console.log(request)
        }
    }

    
}
