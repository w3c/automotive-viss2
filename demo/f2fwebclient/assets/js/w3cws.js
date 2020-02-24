// Vehicle.Body.Lights.IsLowBeamOn,
// Vehicle.Drivetrain.BatteryManagement.BatteryStatus,
// Vehicle.Drivetrain.Transmission.Speed,
// Vehicle.Drivetrain.InternalCombustionEngine.OilGauge,
// Vehicle.Chassis.Axle.Row1.Wheel.Right.Tire.Pressure,
// Vehicle.Chassis.Axle.Row1.Wheel.Right.Tire.PressureLow,
// Vehicle.Chassis.Axle.Row1.Wheel.Right.Tire.Temperature,
// Vehicle.Chassis.Axle.Row1.Wheel.Left.Tire.Pressure,
// Vehicle.Chassis.Axle.Row1.Wheel.Left.Tire.PressureLow,
// Vehicle.Chassis.Axle.Row1.Wheel.Left.Tire.Temperature,
// Vehicle.Chassis.Axle.Row2.Wheel.Right.Tire.Pressure,
// Vehicle.Chassis.Axle.Row2.Wheel.Right.Tire.PressureLow,
// Vehicle.Chassis.Axle.Row2.Wheel.Right.Tire.Temperature,
// Vehicle.Chassis.Axle.Row2.Wheel.Left.Tire.Pressure,
// Vehicle.Chassis.Axle.Row2.Wheel.Left.Tire.PressureLow,
// Vehicle.Chassis.Axle.Row2.Wheel.Left.Tire.Temperature,
// Vehicle.Drivetrain.FuelSystem.Level,
// Vehicle.Body.Outside.Temperature,
// Vehicle.Body.Lights.IsParkingOn,
// Vehicle.Drivetrain.InternalCombustionEngine.EOP,

class W3CClient {
  constructor(debug) {
    self.debug = debug;
  }

  handleRequest(request) {
    switch (request.action) {
      case "get":
        break;
      case "set":
        break;
      case "subscription":
        break;
      default:
    }

    if (self.debug) {
      console.log(request)
    }
  }


}

available_signals = {
  "Speed": "#ws-speedometer"
};

class W3CWebSocket extends W3CClient {
  constructor(debug) {
    super(debug);

  }

  setup(address) {
    alert(address);
    this.socket = new WebSocket(address);
    this.socket.onopen = this.websocket_onopen;
    this.socket.onclose = this.websocket_onclose;
    this.socket.onmessage = this.websocket_onmessage;
  }

  websocket_onopen(event) {
    alert("[open] Connection established");
  };

  websocket_onclose(event) {
    console.error('Socket closed');
  }

  websocket_onmessage(event) {
    var request = JSON.parse(event.data);
    W3CClient.prototype.handleRequest(request) // does nothing atm
    console.log(request.action);
    switch (request.action) {
      case "get":
        console.log(request.requestId);
        console.log(request.subscriptionId);
        console.log(request.value);
        break;
      case "subscription":
        console.log(request.requestId);
        console.log(request.subscriptionId);
        console.log(request.value);
        var obj = reqMap.get(request.requestId);
        obj["subId"] = request.subscriptionId; 
        w3cnotify(request.value, obj["path"] + " : \"" + request.value.toString() + "\"");
        break;
      default:
    }
  }

  requestGet(reqid, path) {
    //{"action":"get", "path":"Vehicle.Cabin.Door.*.*.IsOpen", "requestId":"123"}
    alert("[requestGet] " + path);
    var msg = JSON.stringify({ "action": "get", "path": path, "requestId": reqid });
    this.socket.send(msg);
  }

  requestSet(reqid, path, value) {
    //{"action":"set", "path":"Vehicle.Cabin.Door.Row1.Right.IsOpen", "value":"999", "requestId":"234"}
    this.socket.send(JSON.stringify({ 'action': 'set', 'path': path, 'value': value, 'requestId': reqid }));
  }

  subscribe(reqid, path) {
    //{"action":"subscribe", "path":"Vehicle.Cabin.Door.Row1.Right.IsOpen", "requestId":"234"}
    alert("[subscribe] " + path);
    var msg = JSON.stringify({ "action": "subscribe", "path": path, "filter":"$intervalEQ5", "requestId": reqid });
    this.socket.send(msg);
  }

  unsubscribe(reqid, subscriptionid) {
    //{"action":"unsubscribe", "subscriptionId":"789", "requestId":"234"}
    console.log(reqid, subscriptionid);
    this.socket.send(JSON.stringify({ 'action': 'unsubscribe', 'subscriptionid': subscriptionid, 'requestId': reqid }));
  }
}
