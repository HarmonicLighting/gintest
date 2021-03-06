// Create a socket
var socket;

var signalsArray;

window.onload = function() {

	var button = $("#loginBtn");

	button.click(()=>{
		var user = $("#User").val();
		var pass = $("#Password").val();
		console.log(user);
		console.log(pass);
		var request = new XMLHttpRequest();
		var url = "http://"+window.location.host+"/login";
		var bodyobj = { username:user, password:pass };

		$.ajax({
			url: url,
			type: "POST",
			//contentType: "application/x-www-form-urlencoded",
			data: JSON.stringify(bodyobj),
			contentType: "application/json",
			success: function (data) {
				console.log("Success");
				console.log(data);
				var expirationDate = new Date(data.expire);
				$("#loginStatus").html("Success");
				$("#loginToken").html(`Token <b>${data.token.substring(0, 100)}...</b>`);
				$("#loginTokenExpiration").html(`Expires on ${expirationDate}`);

				if(socket != null && socket.readyState == 1){
					socket.onclose = ()=>{
						socket = new WebSocket(`ws://${window.location.host}/ws?token=${data.token}`);
						socket.onopen = onsocketopen;
						socket.onmessage = onsocketmessage;
						socket.onclose = null;
					};
					socket.close(1000,"The client has re-logged in");
				}else{
					socket = new WebSocket(`ws://${window.location.host}/ws?token=${data.token}`);
					socket.onopen = onsocketopen;
					socket.onmessage = onsocketmessage;
				}
        

			},
			error: function (xhRequest, ErrorText, thrownError) {
				console.warn("Failed to process correctly");
				console.log(xhRequest);
				console.log(ErrorText);
				console.log(thrownError);
				$("#loginStatus").html("Failed");
				$("#loginToken").html("");
				$("#loginTokenExpiration").html("");
        $("#pids-data").empty();
				if(socket != null && socket.readyState == 1){
					socket.close(1000,"The client was not authenticated this time");
				}
			}
		});
	});

};


var onsocketopen = function (event) {
	console.log("Connected!");
	var command = {command:1};
	var jsonCommand = JSON.stringify(command);
	socket.send(jsonCommand);
};

var onsocketmessage = function(event) {


	var message = JSON.parse(event.data);

	if(!message.hasOwnProperty("status")){
		console.warn("This message doesn't include a status field");
		console.log(message);
		return;
	}
	if(message.status < 0){
		console.warn(`This message's status is negative! (${message.status})`);
		console.warn(`Error description: ${message.error}`);
		return;
	}

	switch (message.command) {
	case 1:
		refreshSignals(message);
		break;
	case 4:
		refreshPidValues(message);
		break;
	case 3:
		refreshCurrentUsersCount(message);
		break;
	case 2:
		refresPidListValues(message);
		break;
	default:
		if (message.command < 0) {
			logError(message);
		}else{
			console.warn("Unknown Command "+message.command);
		}
	}
};

var refreshCurrentUsersCount = function(message){
	$("#connectedClients").html(`${message.number}`);
};

// Display a message
var refreshPidValues = function(message) {
	var dt = new Date(message.timestamp/1000000);
	$(`#svalue-${message.index}`).html(`${message.value}`);
	$(`#sstate-${message.index}`).html(`${getSignalStateStr(message.state)}`);
	$(`#sdate-${message.index}`).html(`On ${dt}`);
};

var refresPidListValues = function(message){
	signals = message.pids;

	$("#lastSignalUpdate").html(signals.length);
	for (var i = 0; i < signals.length ; i++){
		signalsArray[signals[i].index].value = signals[i].value;
		signalsArray[signals[i].index].state = signals[i].state;
		signalsArray[signals[i].index].timestamp = signals[i].timestamp;
		if( signals[i].index <100){
			var rmvclass;
			var addclass;
			if(signals[i].state == 1){
				rmvclass = "badSignal";
				addclass = "okSignal";
			}
			var dt = new Date(signals[i].timestamp/1000000);
			$(`#svalue-${signals[i].index}`).html(`${(signals[i].value).toFixed(3)}`);
			$(`#sstate-${signals[i].index}`).html(`${getSignalStateStr(signals[i].state)}`);
			if(signals[i].state == 1){
        $(`#sstate-${signals[i].index}`).removeClass("alert-danger");
        $(`#sstate-${signals[i].index}`).addClass("alert-success");
			}else if(signals[i].state == 2){
        $(`#sstate-${signals[i].index}`).removeClass("alert-success");
        $(`#sstate-${signals[i].index}`).addClass("alert-danger");
			}
			$(`#sdate-${signals[i].index}`).html(`On ${dt}`);
		}
	}
};

var logError = function(message){
	console.warn(`Got command ${message.command}`);
	if(message.status < 0){
		console.warn(`Error in Response Command ${message.command}. Status: ${message.status}, Message: ${message.error} `);
	}
};

// Refresh the signals
var refreshSignals = function(message){
	if(message.status < 0){
		logError(message);
		return;
	}

	//console.log(message);

	signals = message.pids;

	function compare(a,b) {
		if (a.index < b.index)
			return -1;
		if (a.index > b.index)
			return 1;
		return 0;
	}

	signals.sort(compare);

	signalsArray = signals;
	$("#totalSignals").html(signals.length);

	var signalsArea = $("#pids-data");
	signalsArea.empty();
	for (var i = 0; i < signals.length; i++) {

		if( signals[i].index <100){
			signalsArea.append(
				`
          <div class='row'>
            <div class='col-sm-3'>
              <b>${signals[i].name}</b> (${getSignalTypeStr(signals[i].type)}, every ${(signals[i].period / 1000000000).toFixed(2)} s):
            </div>
            <div class='col-sm-2' id='svalue-${signals[i].index}'>
              ${(signals[i].value).toFixed(3)}
            </div>
            <div class='col-sm-2 alert' id='sstate-${signals[i].index}'>
              ${getSignalStateStr(signals[i].state)}
            </div>
            <div class='col-md-5' id='sdate-${signals[i].index}'>
              On ${(new Date(signals[i].timestamp/1000000))}
            </div>
          </div>
          `
			);
		}
	}
};

var getSignalTypeStr = function(sigType){
	var sigtype;
	switch(sigType){
	case 0:
		return "Analogical";
	case 1:
		return "Discrete";
	case 2:
		return "Digital";
	default:
		return `Unknown(${signals[i].type})`;
	}
};

var getSignalStateStr = function(sigState){
	switch (sigState){
	case 0:
		return "Never Updated";
	case 1:
		return "OK";
	case 2:
		return "Bad";
	default:
		return `Unknown (${message.state})`;
	}
};

var requestCommand = function(command){
	obj = {command: command};
	string = JSON.stringify(obj);
	console.log("Sending command ",command);
	console.log(string);
	socket.send(string);
};
