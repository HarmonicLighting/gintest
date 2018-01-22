// Create a socket
var socket = new WebSocket('ws://'+window.location.host+'/ws')


socket.onopen = function (event) {
  //exampleSocket.send("Here's some text that the server is urgently awaiting!");
  console.log("Connected!");
  var command = {command:0}
  var jsonCommand = JSON.stringify(command)
  socket.send(jsonCommand)
};

var updateLastDate = function(date) {
  $(`#lastMessageTime`).html(`${date}`);
}

socket.onmessage = function(event) {

  var dt = new Date(Date.now())
  updateLastDate(dt)

  var message = JSON.parse(event.data)
  switch (message.command) {
    case 0:
      refreshSignals(message.pids)
      break;
    case 1:
      refreshValues(message)
      break;
    default:
      console.warn("The object returned "+message.type);
  }
}

// Display a message
var refreshValues = function(event) {
  var dt = new Date(event.timestamp/1000000);
  $(`#svalue-${event.index}`).html(`${event.value}`);
  $(`#sdate-${event.index}`).html(`On ${dt} -> ${event.timestamp}`);
}

// Refresh the signals
var refreshSignals = function(signals){

    function compare(a,b) {
      if (a.index < b.index)
        return -1;
      if (a.index > b.index)
        return 1;
      return 0;
    }

    signals.sort(compare);

    var signalsArea = $('#pids-data')
    signalsArea.empty();
    for (var i = 0; i < signals.length; i++) {
      //signalsArea.append(`<div class='col-md-4'><b>${signals[i].name}</b> (${signals[i].period / 1000000} ms):</div><div class='col-sm-3' id='s-${signals[i].index}'></div><div class='col-sm-6' id='ss-${signals[i].index}'></div>`)
      signalsArea.append(
        `
        <div class='row'>
          <div class='col-sm-3'>
            <b>${signals[i].name}</b> (${signals[i].period / 1000000} ms):
          </div>
          <div class='col-sm-2' id='svalue-${signals[i].index}'>
          </div>
          <div class='col-sm-2' id='sstate-${signals[i].index}'>
          </div>
          <div class='col-md-5' id='sdate-${signals[i].index}'>
          </div>
        </div>
        `
      );
    }

  }

var exampleSocket = new WebSocket("ws://"+window.location.host+"/ws");


exampleSocket.onmessage = function (event) {
  var obj = JSON.parse(event.data)
  console.log(event.data);
  console.log(obj);
}
