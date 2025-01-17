// HOW to run this file
// 0. npm install eventsource; npm install jsdom
// 1. cd web/static/js
// 2. node test.js
// 3. open http://localhost:3000/test.html


const { JSDOM } = require("jsdom");
const EventSource = require("eventsource");

const url = "http://localhost:5555/sse2";

const dom = new JSDOM();
const window = dom.window;
const document = dom.window.document;
document.body.innerHTML = "<div id='container'></div><div id='messages'></div>";

function printElement(el) {
  console.log("############################");
  console.log("ELEMENT   =>", el);
  console.log("innerHTML   =>", el.innerHTML);
  console.log("outerHTML   =>", el.outerHTML);
  console.log("textContent =>", el.textContent);
  console.log("innerText   =>", el.innerText);
  console.log("outerText   =>", el.outerText);
}

function manipulateElement() {
  let container = document.createElement("div");
  document.body.appendChild(container);
  container.innerHTML = "<p>Hello world</p>";
  let p = document.createElement("p");
  container.appendChild(p);
  let span = document.createElement("span");

  let divId1 = document.createElement("div");
  divId1.id = "div1";
  p.after(span, divId1, document.createElement("div"));
  container.append("TEEEXT");

  printElement(document.body);

  const liElement = `<ul id="list">
  <li><a href="#">Item 1</a></li>
  <li><a href="#">Item 2</a></li>
  <li><a href="#">Item 3</a></li></ul>`;

  let pElement = document.querySelector("div p");
  printElement(pElement);

  let divElement = document.querySelector("div#div1");
  divElement.innerHTML = liElement;
  console.log(divElement.outerHTML);

  let liElements = document.querySelectorAll("ul li");
  console.log(liElements);
  printElement(liElements);

  // Check element is a NodeList
  // console.log("liElements instanceof NodeList===>", liElements instanceof NodeList);
  console.log("liElements instanceof NodeList===>", Array.isArray(liElements));

  let pEl = document.createElement("p");
  pEl.textContent = "Hello World";

  liElements.forEach((li, v, p) => {
    console.log("li.outerHTML===>", li.outerHTML, "; v=>", v, "; text=>", li.textContent);
    li.textContent = "Hello";
    li.append(pEl);
    console.log("li.outerHTML===>", li.outerHTML);
  });

  console.log(divElement.outerHTML);
}

manipulateElement();

function processTestSSE(url) {
  console.log("<<<<<<<<< processSSE >>>>>>>>>>>>");
  const eventSource = new EventSource("/sse");

  eventSource.onopen = function (event) {
    console.log("SSE connection opened");
  };

  eventSource.onerror = function (event) {
    console.error("SSE error:", event);
  };

  eventSource.onmessage = function (event) {
    console.log("SSE message received:", event.data);
    // Do something with the received data
  };

  eventSource.addEventListener("custom-event", function (event) {
    console.log("Custom SSE event received:", event.data);
    // Do something with the received data
  });

  eventSource.addEventListener("ping", function (event) {
    console.log("SSE message received:", event.data);
    // Do something with the received data
    const outputDiv = document.getElementById("output");
    outputDiv.innerHTML += "Ping received<br>";
  });
}

// processTestSSE(url);

console.log("<<<<<<<<< ############################ >>>>>>>>>>>>");

function processSSE(url, messageCallback, errorCallback) {
  const eventSource = new EventSource(url);

  window.addEventListener("beforeunload", function (event) {
    eventSource.close();
  });

  eventSource.onopen = function (event) {
    console.log("SSE connection opened");
  };

  eventSource.onerror = function (event) {
    errorCallback(eventSource, event);
  };

  // Listen for messages from the server.
  eventSource.onmessage = function (event) {
    messageCallback(event);
  };

  eventSource.addEventListener("aloha", function (event) {
    console.log("Custom SSE event received:", event);
    // Do something with the received data
  });

  return eventSource;
}

function handleNewMessage(event) {
  console.log(" handleNewMessage >>>", event);
  const messageContainer = document.getElementById("messages");
  const newMessage = document.createElement("div");
  newMessage.textContent = JSON.parse(event.data).date;
  messageContainer.appendChild(newMessage);
  console.log(" handleNewMessage <<<", messageContainer.outerHTML);
}

// Function to handle errors
function handleError(es, event) {
  console.error("SSE error:", event);
  es.close();
}
console.log("<<<<<<<<< ############################ >>>>>>>>>>>>");

// const eventSource = processSSE(url, handleNewMessage, handleError);
