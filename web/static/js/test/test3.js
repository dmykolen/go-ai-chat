const { JSDOM } = require("jsdom");

const pageFromHTML1 = JSDOM.fromFile("/Users/dmykolen/go/src/go-ai/web/static/js/test/index.html");
pageFromHTML1
  .then((dom) => {
    console.log(dom.window.document.body, "\n\n\n");
    b = dom.window.document.body;
    console.log(b.innerHTML);

    chatsStart = dom.window.document.querySelectorAll(".chat-start");
    console.log(chatsStart);
    console.log(chatsStart.length);
    chatsStart.forEach((chat) => {
      console.log(chat);
      chat.addEventListener("click", function (event) {
        console.log("click-evt:", event, "; THIS==>", this.outerHTML);
      });
      chat.classList.add("bg-red-500");
      chat.appendChild(dom.window.document.createElement("div"));
    });
    chatsStart.forEach((chat) => {
      chat.click();
    });
    console.log(dom.window.document.getElementById("chat").outerHTML);
  })
  .then(() => {
    console.log("DONE");
  });
