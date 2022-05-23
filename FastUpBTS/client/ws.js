var wsuri = "ws://localhost:8080";
var sock = null;
var testing = 0;
var payload = "";
var payloadlen = 1024*1024*1; // 1m

onmessage = function (event) {
	console.log('Received message ' + event.data);

	if (event.data=="init") {
		for (i = 0; i < payloadlen; i++) {
			payload += "a";
		}
	}

	if (event.data=="start") {
		if (testing==1) {
			console.log("test running");
			return
		}
		console.log("start test in worker");
		console.log("payload length", payload.length);
        testing = 1;
		sock = new WebSocket(wsuri);

		sock.onopen = function() {
			console.log("connected to " + wsuri);
			testing = 1;
			t = 0;
			sock.send("start");

			const fun = () => {
				if (testing!=1 || sock.readyState!=1) {
					return
				}
				if (sock.bufferedAmount < payloadlen) {
					sock.send(payload);
					t++;
				}
				if (testing==1 && sock.readyState==1) {
					setTimeout(() => {
						fun();
					}, 5)
				}
			}
			
			fun();
		}

		sock.onclose = function(e) {
			testing = 0;
			console.log("connection closed (" + e.code + ")");
		}

		sock.onmessage = function(e) {
			console.log("message received: " + e.data);
			if (e.data.substring(0,6)=="Result") {
				results = e.data.substring(7).split(",");
				postMessage({type:"results", args:results});
			}
		}
		
	}
	if (event.data=="close sock") {
		sock.close();
	}
}