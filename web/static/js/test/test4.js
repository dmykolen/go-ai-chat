const { JSDOM } = require("jsdom");

const dom = new JSDOM();
const window = dom.window;
const document = dom.window.document;

const chatMsg = document.createElement("div");
const chatTypes = ["chat-start", "chat-end"];

function addChat(value, type) {
  const chat = document.createElement("div");
  chat.classList.add("chat", chatTypes[type]); // chat-start=0, chat-end=1
  const chatBubble = document.createElement("div");
  chatBubble.innerHTML = value;
  // chatBubble.innerHTML = value.replace(/\\n/g, "<br>").replace(/\\t/g, "\t");
  // chatBubble.classList.add("chat-bubble", type === 0 ? "chat-bubble-primary" : "chat-bubble-error");
  // chatBubble.classList.add("chat-bubble", type === 0 ? "chat-bubble-primary" : "chat-bubble-error", "overflow-x-auto", "text-sm", "whitespace-pre");
  chatBubble.classList.add(
    "chat-bubble",
    "whitespace-pre-wrap",
    type === 0 ? "chat-bubble-primary" : ("hover:shadow-indigo-500/40", "shadow-lg", "shadow-rose-900/10", "text-sm", "whitespace-pre-wrap")
  );

  if (value == "") {
    chatBubble.appendChild(createSpinnerSpan(1));
  }
  chat.appendChild(chatBubble);
  chatMsg.appendChild(chat);
}

addChat("1st chat", 0);
addChat("2nd chat", 1);
addChat("3rd chat", 0);

console.log(chatMsg.outerHTML);
