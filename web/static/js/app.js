// @ts-nocheck

// const { mdParse } = require("./main");

const chatMsg = document.getElementById("chat-msg");
const chatSI = document.getElementById("chat-send-input");
const loadingTypes = ["loading-spinner", "loading-dots", "loading-ball", "loading-ring", "loading-bars", "loading-infinity"];
const chatTypes = ["chat-start", "chat-end"]; // for input to addChat()
const preloadedImage = new Image();
preloadedImage.src = '/img/robot_image_3.png';
let currentTabIndex = 0;
let totalTabs = 0;
const tabID = getConnectionId();
const sseUrl = `/sse2?connectionId=${tabID}`;

const chatFooterRating = `<div class="chat-footer place-items-end">
        <time class="align-top opacity-50 text-base-content/50 text-xs">${new Date().toLocaleDateString()}</time>
        <form class="-mt-2 flex" hx-post="/api/v1/rate" hx-trigger="click" hx-include="[name=chatRating],#id-chat-input" hx-swap="none" hx-vals="js:{chatIdx: evalHTMX_rating(event)}" hx-ext="json-enc">
            <input type="radio" name="chatRating" class="btn btn-sm mask mask-star-2 bg-secondary/20 hover:bg-indigo-500 hover:scale-125" value="1" onclick="starClick($(this))">
            <input type="radio" name="chatRating" class="btn btn-sm mask mask-star-2 bg-secondary/20 hover:bg-indigo-500 hover:scale-125" value="2" onclick="starClick($(this))">
            <input type="radio" name="chatRating" class="btn btn-sm mask mask-star-2 bg-secondary/20 hover:bg-indigo-500 hover:scale-125" value="3" onclick="starClick($(this))">
            <input type="radio" name="chatRating" class="btn btn-sm mask mask-star-2 bg-secondary/20 hover:bg-indigo-500 hover:scale-125" value="4" onclick="starClick($(this))">
            <input type="radio" name="chatRating" class="btn btn-sm mask mask-star-2 bg-secondary/20 hover:bg-indigo-500 hover:scale-125" value="5" onclick="starClick($(this))">
        </form>
    </div>`

    const senderIdKey = "_sender_id_"
    const tabSenderId = getSenderId();
    const bc = new BroadcastChannel('message_broadcast')
    bc.onmessage = handleBCMessage;


  function getSenderId() {
      let senderId = sessionStorage.getItem(senderIdKey) || getUniqueId();
      sessionStorage.setItem(senderIdKey, senderId);
      return senderId;
  }
  function getUniqueId() {
      return '_' + Math.random().toString(36).substr(2, 9);
  }
  function makeMessage(message) {
      return {
          id: getUniqueId(),
          senderId: tabSenderId,
          message: message,
      };
  }

  function handleBCMessage(event) {
    console.log('Receive BC msg =>', event);
}
function postBCMessage(msg) {
    let m = makeMessage(msg);
    console.log('Send BC message: ', m)
    bc.postMessage(msg);
}

    function initializeTab() {
      // Increment total tabs counter in localStorage
      console.warn('Initialize tab!')
      totalTabs = parseInt(localStorage.getItem('total-tabs')) || 0;
      totalTabs++;
      localStorage.setItem('total-tabs', totalTabs);

      // Assign currentTabIndex in sessionStorage
      currentTabIndex = totalTabs;
      sessionStorage.setItem('current-tab-index', currentTabIndex);
      console.log(`Tab opened with index: ${sessionStorage.getItem('current-tab-index')}`);
      console.log('sessionStorage:', sessionStorage);
  }

  function handleTabClose() {
    // Decrement total tabs counter in localStorage
    let totalTabs = parseInt(localStorage.getItem('total-tabs')) || 0;
    if (totalTabs > 0) {
        totalTabs--;
        localStorage.setItem('total-tabs', totalTabs);
    }

    // Remove currentTabIndex from sessionStorage
    sessionStorage.removeItem('current-tab-index');
    console.log(`Tab closed. Remaining tabs: ${totalTabs}`);
    es?.close();
}

function getConnectionId() {
  let connectionId = sessionStorage.getItem('connectionId');
  if (!connectionId) {
      connectionId = Math.random().toString(36).substring(2, 10) + Math.random().toString(36).substring(2, 10);
      sessionStorage.setItem('connectionId', connectionId);
  }
  console.log(`Get current tabID = ${connectionId}`)
  return connectionId;
}

window.addEventListener('load', initializeTab);
window.addEventListener('beforeunload', handleTabClose);

function starClick(e){
  console.log('Set stars on msg!',e)
  e.prevAll().switchClass("bg-secondary/20 border-2", "bg-secondary", 500, "easeInOutQuad");
  e.nextAll().switchClass("bg-secondary border-2", "bg-secondary/20", 500, "easeInOutQuad");
  e.switchClass("bg-secondary/20", "bg-secondary border-2", 500, "easeInOutQuad");
}

function evalHTMX_rating(e) {
  console.info('START /rating HTMX request! Evt =>', e)
  // Find the closest parent element with the class 'chat-end'
  let chatEndElement = e.target.closest('.chat-end');

  // If no such element is found, return -1 (indicating an error or not found)
  if (!chatEndElement) {
      return -1;
  }
  let allChatEnds = Array.from(document.querySelectorAll('.chat'));
  let index = allChatEnds.indexOf(chatEndElement);

  console.log(`Total 'chat-end' elements = ${allChatEnds.length}; IDX_CURRENT=${index}`)
  return index;
}

function userAuthCheck(es) {
  if (document.cookie.includes("username") && document.cookie.includes("userId")) {
    console.log("userAuthCheck(): User is authenticated!");
    $("#login-li").hide();
    $("#logout-li").removeClass("hidden");

    processSSE(`/sse2?connectionId=${tabID}`, handleEventChatGPT, handleErrorSSE);
  } else {
    console.log("userAuthCheck(): User is not authenticated!");
    $("#login-li").show();
    $("#logout-li").addClass("hidden");
    if (es != null) {
      es.close();
      es = null;
    }
  }
}

function sideBar() {
  $("#drawer-toggle").prop("checked", false);
  $(document).mousemove(function (event) {
    if ($("#loginModal").length > 0 && !$("#loginModal").hasClass("hidden")) {
      return;
    }
    screenWidth = $(window).width();
    if (event.pageX < screenWidth / 50) {
      if (!$("#drawer-toggle").prop("checked")) {
        console.debug("Hover was on the left side of the screen. event.pageX:", event.pageX, "; screenWidth:", screenWidth);
        $("#drawer-toggle").prop("checked", true);
      }
    } else if (event.pageX > screenWidth / 3) {
      if ($("#drawer-toggle").prop("checked")) {
        console.debug("Sidebar->CLOSE. event.pageX:", event.pageX, "; screenWidth:", screenWidth);
        $("#drawer-toggle").prop("checked", false);
      }
    }
  });
}

function controlTextArea() {
  keyPrevious = "";
  const resizeInput = (evt) => {
    inp = evt.target;
    if ((keyPrevious == "Control" || keyPrevious == "Meta") && evt.key === "Enter") {
      console.log("Enter key pressed");
      $("#chat-send").click();
    }
    keyPrevious = evt.key;
    inp.style.height = inp.scrollHeight < parseInt(inp.style.height) ? "" : inp.scrollHeight + "px";
    console.debug("input:TextArea: AFTER : key", evt.key, "height=", inp.style.height, " clientHeight=", inp.clientHeight, " ScrollHeigt=", inp.scrollHeight);
  };

  $("#chat-send-input")
    .keydown(resizeInput)
    .on("paste", (e) => {
      console.debug("Pasted content: ", (e.originalEvent.clipboardData || window.clipboardData).getData("text"));
      setTimeout(() => {
        resizeInput(e);
      }, 100);
    })
    .focusout(function () {
      console.debug("FOCUS_OUT=>");
      $(this).animate({ height: "" }, 300, "linear", () => {
        console.debug("Animation complete.");
      });
    })
    .focusin(function () {
      const el = $(this)[0];
      console.debug("FOCUS_IN=> height=", el.style.height, "; clientHeight=", el.clientHeight, "; scrollHeight=", el.scrollHeight);
      $(this).animate({ height: el.scrollHeight + "px" }, 500, "linear");
    });
}

// create and return a new span element
function createSpinnerSpan(type) {
  const span = document.createElement("span");
  span.classList.add("loading", loadingTypes[type], "loading-sm");
  return span;
}

function printLog(el) { console.debug("element size:", el.style.height, "; scrollTop:", el.scrollTop, "; clientHeight:", el.clientHeight, "; scrollHeight:", el.scrollHeight, "; offsetHeight:", el.offsetHeight); }

function addListener_btn_send() {
  document.getElementById("chat-send")?.addEventListener("click", (event) => {
    processSSE(sseUrl, handleEventChatGPT, handleErrorSSE);
    countClick("btnChatSend");
    printLog(chatMsg);
    console.log("user message: ", chatSI.value);
    // @ts-ignore
    addChat(chatSI.value, 0);
    chatSI.animate({ height: "" }, 300, "linear", () => {
      console.log("Animation complete.");
    });
    addChat("", 1);
    printLog(chatMsg);
    chatMsg.scrollTo(0, chatMsg.scrollHeight, { behavior: "smooth" });
  });
}

function createAvatar(type) {
  // Create the main container div with classes "chat-image" and "avatar"
  const chatImageDiv = document.createElement("div");
  chatImageDiv.classList.add("chat-image", "avatar");

  const innerDiv = document.createElement("div");
  innerDiv.classList.add("w-10", "rounded-full", "border-2", "border-accent", "text-4xl");
  // innerDiv.textContent = type == 0 ? "🥷🏽" : "🤖";
  if (type == 0) {
    innerDiv.textContent = "🥷🏽";
  } else {
        const imgElement = document.createElement('img');
        // imgElement.src = '/img/robot_image_3.png';
        imgElement.src = preloadedImage.src;
        innerDiv.appendChild(imgElement);
  }

  chatImageDiv.appendChild(innerDiv);
  return chatImageDiv;
}

function createChatHeader(type) {
  const chatHeader = document.createElement("div");
  chatHeader.classList.add("chat-header", "font-bold", "text-base-content/50");
  chatHeader.textContent = type == 0 ? "user" : "assistant";
  return chatHeader;
}

function chatBubbleInnerForUser(value) {
  return `<div class="bg-clip-text bg-gradient-to-tr font-bold from-primary text-transparent to-secondary whitespace-pre-wrap">${value}</div>`
}

// type=0 - chat-start, type=1 - chat-end, ...
function addChat(value, type) {
  const chat = document.createElement("div");
  chat.classList.add("chat", chatTypes[type]); // chat-start=0, chat-end=1

  chat.appendChild(createAvatar(type));
  chat.appendChild(createChatHeader(type));

  const chatBubble = document.createElement("div");
  chatBubble.innerHTML = type == 0 ? chatBubbleInnerForUser(value) : value;
  // chatBubble.innerHTML = value.replace(/\\n/g, "<br>").replace(/\\t/g, "\t");

  $(chatBubble).addClass(type === 0 ? "chat-bubble chat-bubble-primary whitespace-pre-wrap bg-black text-accent min-h-[2rem]" : "chat-bubble prose");

  if (value == "") {
    chatBubble.appendChild(createSpinnerSpan(1));
  }
  chat.appendChild(chatBubble);
  console.log(">>> ADD chat:", chat);
  chatMsg.appendChild(chat);
  $(chatMsg.parentElement).animate({scrollTop: $('.chat:last').position().top}, 2000, 'easeOutBounce')

  if (type == 1) {
    $(chat).append(chatFooterRating);
    htmx.process($(chat).find('.chat-footer>form')[0]);
  } else {
    $(chat).append(`<div class="chat-footer"><time class="align-top opacity-50 text-base-content/50 text-xs">${new Date().toLocaleString('sv-SE')}</time></div>`);
  }
}

function processSSE(url, messageCallback, errorCallback) {
  if (es == null || es.readyState == 2) {
    console.log("SSE is not active, init new one");
    es = new EventSource(url);
  } else {
    console.log("SSE is already active, do nothing");
    return es;
  }

  window.addEventListener("beforeunload", function (event) {
    es.close();
    localStorage.setItem('sse-active', 'false')
  });

  window.addEventListener('storage', (event) => {
    if (event.key === 'sse-active' && event.newValue === 'false') {
      console.warn('SSE inited on another tab');
      // es.close();
    }
  });

  es.onopen = function (event) {
    console.log("SSE connection opened");
  };

  es.onerror = function (event) {
    console.log("onerror SSE event received:", event);
    errorCallback(es, event);
  };

  // Listen for messages from the server.
  es.onmessage = function (event) {
    console.debug("onmessage SSE event received:", event.data);
  };

  es.addEventListener("chatgpt_response", function (event) {
    console.debug("Custom SSE event received:", event);
    messageCallback(event);
  });

  es.addEventListener("sql_table_as_json", function (event) {
    console.debug("Custom SSE event [sql_table_as_json] received:", event);
    messageCallback(event);
  });

  sseLogStatus();
  localStorage.setItem('sse-active', 'true')

  // check after 5 sec if SSE is active
  setTimeout(() => {
    sseLogStatus();
  }, 5000);
  return es;
}

function sseLogStatus() {
  switch (es?.readyState) {
    case 0:
      console.warn("The connection has not yet been established.");
      break;
    case 1:
      console.warn("The connection is established and communication is possible.");
      break;
    case 2:
      console.warn("The connection is going through the closing handshake.");
      break;
  }
}

let msgArr = [];
let isTypingCompleted = true;
let queuedText = "";
let tp;
let textChunks = "";

// handle new msg from sse dependent on event type
function handleEventChatGPT(e) {
  console.debug("<<handleEventChatGPT>> eventType:", e.type, "; data:[" + e.data + "]");

  const chatMsg = document.getElementById("chat-msg");
  const lastChild = $("#chat-msg").children().last();
  let lastBubble = null;

  if (lastChild.hasClass("chat-end")) {
    lastBubble = lastChild.find(".chat-bubble");
  } else {
    // Last child does not have 'chat-end' class, create a new one
    const newChatEnd = $('<div class="chat chat-end"></div>');
    lastBubble = $('<div class="chat-bubble chat-bubble-error"></div>');
    newChatEnd.append(lastBubble);
    $("#chat-msg").append(newChatEnd);
    console.debug(`<<handleEventChatGPT>> Add new element with .chat-end, after el .chat-bubble to it!\nChatEnd====> [${newChatEnd}]\nlastBubble====> [${lastBubble}]`);
  }

  chatMsg.scrollTo(0, chatMsg.scrollHeight, { behavior: "smooth" });

  console.debug("<<handleEventChatGPT>> Choose typeing effect! location.pathname:", location.pathname);
  if (location.pathname == "/voip" || location.pathname == "/aidb") {
    // typingEffectVerySmallParts(e.data.toString().replace(/\\n/g, "<br>").replace(/\\t/g, "\t"));
    typingEffectVerySmallParts(e.data.toString());
  } else {
    typingEffect(e.data.toString().replace(/\\n/g, "<br>").replace(/\\t/g, "\t"));
  }
}

function chatBeautifullLast() {
  $('.chat.chat-end>div.chat-bubble:last').each((i, v) => {
    console.log(`parseMdToHTML: INPUT ELEMENT[idx=${i}]; Text=>[${$(this).text()}]`);
    console.log(`parseMdToHTML: TEXT::::>`, textChunks);
    if (isValidJSON(textChunks)) {
      console.log('Start parse JSON!');
      addTableToChatFromJSON($(v), textChunks);
    } else {
      v.innerHTML = mdParse(textChunks);
      console.log(`parseMdToHTML: INPUT ELEMENT[idx=${i}] : ${v.innerHTML}`);
    }
  });
  textChunks = "";
}

function typingEffectVerySmallParts(text) {
  if (text == "######") {
    console.log('SSE Answer complete! Got last part of answer!')
    $("span.loading.loading-dots").remove();
    chatBeautifullLast();
    $('.chat.chat-end>div:last>time').text(new Date().toLocaleString('sv-SE'))
    return;
  }
  textChunks += `${text}\n`;
  $(".chat-end > .chat-bubble > span.loading").last().before(text);
}

function typingEffect(text) {
  console.log("<<typingEffect>> isTypingCompleted=", isTypingCompleted, "; queuedText=", msgArr, "; text=", text);

  if (msgArr.length == 0 && text == "######") {
    $("#type").contents().unwrap();
    $("span.loading.loading-dots").remove();
    return;
  }

  if (isTypingCompleted) {
    $("#type").contents().unwrap();
    if ($("#type").length == 0) {
      $(".chat-end > .chat-bubble > span.loading").last().before('<span id="type"></span>');
    }

    isTypingCompleted = false;
    tp = new Typed("#type", {
      strings: [text],
      typeSpeed: 5,
      showCursor: false,
      contentType: "html",
      onComplete: function () {
        isTypingCompleted = true;
        if (msgArr.length > 0) {
          typingEffect(msgArr.shift());
        }
      },
    });
  } else {
    console.log("<<typingEffect>> push text to msgArr");
    msgArr.push(text);
  }
}

// Function to handle errors
function handleErrorSSE(es, event) {
  console.error("SSE error:", event);
  es.close();
}

function countClick(elemName) {
  const storageKey = `click_${elemName}`;
  localStorage.setItem(storageKey, localStorage.getItem(storageKey) ? Number(localStorage.getItem(storageKey)) + 1 : 1);
  console.log(`Click count for ${elemName}: ${localStorage.getItem(storageKey)} times`);
}

function random(number) {
  return Math.floor(Math.random() * number);
}

function randomColor() {
  return `rgb(${random(255)} ${random(255)} ${random(255)})`;
}

function errorShowOnBodyCenter(text) {
  $("body").append(`<div id="error-board" class="hidden absolute rounded-md w-2/3 h-auto max-h-[23%] max-w-[44%] left-[28%] top-[40%] p-4 bg-red-700 shadow-lg shadow-red-900 overflow-auto"><span class="indicator">🤬</span><p class="flex flex-col font-mono text-center text-lg text-white"><span class="-mt-7 text-center text-xl font-extrabold mb-10">ERROR</span>${text}</p></div>`);
  $("#error-board").toggle("puff", {}, 500);
  $('body').on('click',()=>{errorRemoveByTimer(50)})
}

/**
 * @param {number} millisTimeOut
 */
function errorRemoveByTimer(millisTimeOut) {
  setTimeout(() => {
    $("#error-board").toggle("puff", {}, 500, () => {
      $("#error-board").remove();
    });
  }, millisTimeOut);
}

function calculateCenterPosition($element) {
  const centerX = ($(window).width() / 2) - ($element.outerWidth() / 2);
  const centerY = ($(window).height() / 2) - ($element.outerHeight() / 2);
  console.log(`Element: Width: ${$element.outerWidth()}, Height: ${$element.outerHeight()}; Calculate screen center: X=${centerX},  Y=${centerY}`)
  return { centerX, centerY };
}

function moveElemToCenterAndViseVersa($element) {
  const { centerX, centerY } = calculateCenterPosition($element);
  const currentX = $element.offset().left;
  const currentY = $element.offset().top;
  console.log(`Current X: ${currentX}, Current Y: ${currentY}`)

  const translateX = centerX - currentX;
  const translateY = centerY - currentY;
  console.log(`Translate X: ${translateX}, Translate Y: ${translateY}`)

  $element.toggleClass('absolute');
  const transformValue = $element.css('transform') === 'none' ? `translate(${translateX}px, ${translateY}px) rotate3d(1, 2, 1, 360deg) scale(1.5)` : '';
  $element.css('transform', transformValue);
}

