// Create a socket
var socket = new WebSocket('ws://'+window.location.host+'/ws')


socket.onopen = function (event) {
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

  //console.log(message);


  switch (message.command) {
    case 0:
      refreshSignals(message)
      break;
    case 1:
      refreshPidValues(message)
      break;
    case 2:
      refreshCurrentUsersCount(message)
      break;
    case 3:
      refresPidListValues(message)
      break;
    default:
      if (message.command < 0) {
        logError(message)
      }else{
        console.warn("Unknown Command "+message.command);
      }
  }
}

var refreshCurrentUsersCount = function(message){
  $('#connectedClients').html(`${message.number}`);
}

// Display a message
var refreshPidValues = function(message) {
  var dt = new Date(message.timestamp/1000000);
  $(`#svalue-${message.index}`).html(`${message.value}`);
  $(`#sstate-${message.index}`).html(`${getSignalStateStr(message.state)}`);
  $(`#sdate-${message.index}`).html(`On ${dt}`);
}

var refresPidListValues = function(message){
  console.log(message);
  signals = message.pids
  for (var i = 0; i < signals.length; i++) {
    var dt = new Date(signals[i].timestamp/1000000);
    $(`#svalue-${signals[i].index}`).html(`${signals[i].value}`);
    $(`#sstate-${signals[i].index}`).html(`${getSignalStateStr(signals[i].state)}`);
    $(`#sdate-${signals[i].index}`).html(`On ${dt}`);
  }
  console.log("Updated ",signals.length, " signals");
}

var logError = function(message){
  if(message.status < 0){
    console.warn(`Error in Response Command ${message.command}. Status: ${message.status}, Message: ${message.error} `);
  }
}

// Refresh the signals
var refreshSignals = function(message){
  if(message.status < 0){
    logError(message)
    return
  }

  //console.log(message);

  signals = message.pids

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

      signalsArea.append(
        `
        <div class='row'>
          <div class='col-sm-3'>
            <b>${signals[i].name}</b> (${getSignalTypeStr(signals[i].type)}, every ${(signals[i].period / 1000000000).toFixed(2)} s):
          </div>
          <div class='col-sm-2' id='svalue-${signals[i].index}'>
            ${(signals[i].value).toFixed(2)}
          </div>
          <div class='col-sm-2' id='sstate-${signals[i].index}'>
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

  var getSignalTypeStr = function(sigType){
    var sigtype
    switch(sigType){
      case 0:
        return 'Analogical'
      case 1:
        return 'Discrete'
      case 2:
        return 'Logic'
      default:
        return `Unknown(${signals[i].type})`
    }
  }

  var getSignalStateStr = function(sigState){
    switch (sigState){
      case 0:
        return 'Never Updated'
      case 1:
        return 'OK'
      case 2:
        return 'Bad'
      default:
        return `Unknown (${message.state})`
    }
  }

  var requestCommand = function(command){
    obj = {command: command}
    string = JSON.stringify(obj)
    console.log("Sending command ",command);
    console.log(string);
    socket.send(string)
  }
