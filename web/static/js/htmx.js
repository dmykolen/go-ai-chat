// @ts-nocheck

$(function () {
  // htmx.logAll();
  htmx.config.historyCacheSize = 30;
  htmx.config.historyEnabled = true;
  htmx.config.defaultSwapDelay = 50;
  htmx.config.defaultSettleDelay = 100;
  htmx.config.globalViewTransitions = true;

  console.log("htmx.config => ", htmx.config);
  htmx.logger = function (elt, event, data) {
    if (console) {
      console.debug('<<HTMX_LOG>> Event: %o\nData: %o\nElement: %o', event, data, elt);
    }
  };
});

document.body.addEventListener("htmx:configRequest", function (evt) {
  console.log("HTMX_LOG<configRequest> evt.detail:", evt.detail);
  $("#chat-send-input")
    .animate({ height: "0px" }, "slow", () => {
      console.log("Set '0px' height for #chat-send-input finish!");
    })
    .val("");
});

document.body.addEventListener("htmx:afterRequest", function (e) {
  console.debug("HTMX_LOG<afterRequest> e:", e);
  if (e.detail.xhr.status === 302) {
    var redirectUrl = e.detail.xhr.getResponseHeader('Location');
    console.warn(`During HTMX request ${e.detail.requestConfig.path} got 302 redirect to ${redirectUrl}`)
    if (redirectUrl && e.detail.requestConfig.path.includes("/api/v1/users")) {
        window.location.href = redirectUrl;
    }
  }


  const det = e.detail;
  console.log(
    `<htmx:afterRequest> targetId=> ${det.target.id}; detail.xhr.status=> ${det.xhr.status}; requestBody=> ${det.requestConfig.parameters}; responseText=> ${det.xhr.responseText}`
  );

  if (det.target.id === "chat-msg" && det.xhr.status == 200) {
    if (!isValidJSON(det.xhr.responseText)) {
      console.info('htmx target was chat-msg, but response NOT json!')
      return;
    }
    const resp = JSON.parse(det.xhr.responseText);
    if (resp.status == "OK") {
      console.log("HTMX_LOG<afterRequest> Request to LLM was successful!");
      if (resp.chatId) {
        console.info('>>>> SET chat id=[%s]', resp.chatId)
        $("#id-chat-input").val(resp.chatId);
      }
    }
  } else if (det.target.id === "loginModal" && det.xhr.status == 200) {
    const resp = JSON.parse(det.xhr.responseText);
    if (resp.status == "OK") {
      console.log("HTMX_LOG <afterRequest> Login successful!");
      $("#loginModal").toggleClass("hidden");
      $("#login-li").hide();
      $("#logout-li").removeClass("hidden");
      es?.close();
      processSSE("/sse2", handleEventChatGPT, handleErrorSSE);
    }
  }
});
document.body.addEventListener("htmx:sendError", function (e) {
  console.log("HTMX_LOG <sendError> =>", e.detail.error);
  processErr(e);
});
document.body.addEventListener("htmx:responseError", function (evt) {
  console.error("HTMX_LOG <responseError> =>", evt.detail);
  if (evt.detail.xhr.status === 401) {
    console.log('Got http_code=401 from BE')
    // window.location.href = "/login";
  }
  if (evt.detail.target.id === "chat-msg") {
    processErr(evt);
  } else {
    errorShowOnBodyCenter(`HTMX::ERROR => ${evt.detail.error}`)
  }
});

function processErr(evt) {
  $("#chat-msg .chat-end:last").remove();
  const errAlert = document.createElement("div");
  errAlert.classList.add("flex", "justify-center");
  errAlert.innerHTML = `
      <div class="bg-red-100 text-red-700 border border-red-700 rounded px-4 py-3 leading-normal mb-4">
        Error sending message! HTTP Status: ${evt.detail.xhr.status}
      </div>
    `;
  evt.detail.target.appendChild(errAlert);
  setTimeout(() => {
    evt.detail.target.removeChild(errAlert);
  }, 3000);
}
