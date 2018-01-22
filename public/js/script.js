var exampleSocket = new WebSocket("ws://"+window.location.host+"/ws");

exampleSocket.onopen = function (event) {
  //exampleSocket.send("Here's some text that the server is urgently awaiting!");
  console.log("Connected!");
};

exampleSocket.onmessage = function (event) {
  var obj = JSON.parse(event.data)
  console.log(event.data);
  console.log(obj);
}

exampleSocket.onclose = function (){
  exampleSocket.close()
   exampleSocket = new WebSocket("ws://"+window.location.host+"/ws");  
}
