document.getElementById("audioFile2").addEventListener("change", function () {
  const file = this.files[0];
  if (file) {
    // btn-disabled
    document.getElementById("audioPlayerDiv").classList.remove("hidden");
    $("#btnSubmitV2T").removeClass("btn-disabled");
    const objectURL = URL.createObjectURL(file);
    document.getElementById("audioPlayer").src = objectURL;
  }
});

document.addEventListener("submit", function (e) {
  if (e.target.id === "uploader1") {
    console.log("<<Event-Submit>> target_id=", e.target.id, "\nEVENT:", e);
    if (document.getElementById("audioFile").files.length > 0) {
    }
  }
});

var audio = new Audio("test.mp3");

function playAudio() {
  audio.play();
}

function pauseAudio() {
  audio.pause();
}

document.body.addEventListener("htmx:afterOnLoad", function (evt) {
  console.debug("EVENT-htmx:afterOnLoad! event:", evt);
  const det = evt.detail;
  const params = det.requestConfig.parameters;
  console.log("new_elem:", evt.target, "\nisFailed:", det.failed, "\ntargetId=>", det.target.id, "\ndetail.xhr.status=>", det.xhr.status, "\nparams=>", params);
  if (det.xhr.status == 200 && det.target.id === "uploader1") {
    const resp = JSON.parse(det.xhr.responseText);
    console.log("Resp:", resp.status, "File upload is complete. File:", params.file[0]);
    if (params.file[0]) {
      // downloadLink = document.querySelector("#audioPlayer > a")
      const fig = document.getElementById("fig1");
      const downloadLink = document.createElement("a");
      downloadLink.textContent = "Download your audio";
      downloadLink.setAttribute("href", "/uploads/" + params.file[0].name);
      downloadLink.setAttribute("download", "NEW_" + params.file[0].name);
      fig.appendChild(downloadLink);
    }
  }
});

function InitSTT() {
  console.log("InitSTT()");
  let mediaRecorder;
  let audioChunks = [];
  let audioFile;

  document.body.addEventListener("htmx:configRequest", function (e) {
    console.log("htmx:configRequest => ELEMENT:", $("#audioFile")[0]);
    if ($("#audioFile")[0].files.length > 0) {
      e.detail.parameters.file = $("#audioFile")[0].files[0];
    } else {
      e.detail.parameters.file = audioFile;
    }
    console.log("htmx:configRequest => add file. Event:", e);
  });

  document.body.addEventListener("htmx:afterOnLoad", function (e) {
    $("#resultSTT").removeClass("skeleton");
  });

  // Getting user media
  navigator.mediaDevices
    .getUserMedia({ audio: true })
    .then((stream) => {
      mediaRecorder = new MediaRecorder(stream);
      mediaRecorder.ondataavailable = (event) => {
        audioChunks.push(event.data);
      };
      mediaRecorder.onstop = () => {
        console.log("Recording stopped. Audio chunks", audioChunks);
        const audioBlob = new Blob(audioChunks, { type: "audio/wav" });
        audioFile = new File([audioBlob], "audio.wav", {
          type: "audio/wav",
        });
        addNewAudio(audioBlob);
        audioChunks = [];
      };
      mediaRecorder.onstart = () => {
        console.log("Recording started");
        $("#ac").empty();
      };
    })
    .catch((err) => {
      console.log("Error: " + err);
    });

  document.getElementById("audioFile").addEventListener("change", function () {
    const file = this.files[0];
    if (file) {
      $("#btnSubmitV2T").removeClass("btn-disabled");
      addNewAudio(file);
    }
  });

  function addNewAudio(blob) {
    $("#ac").empty();
    let audio = new Audio(URL.createObjectURL(blob));
    audio.id = "acAudio";
    audio.controls = true;
    audio.autoplay = true;
    audio.volume = 0.5;
    console.log("Add audio:", audio);
    $("#ac").append(audio);
    $("#ac").slideDown("slow");
  }

  function toggleBtns(isDisabledStart, isDisabledPause, isDisabledStop) {
    $("#startBtn").prop("disabled", isDisabledStart);
    $("#pauseBtn").prop("disabled", isDisabledPause);
    $("#stopBtn").prop("disabled", isDisabledStop);
  }

  function addDownload(blob, name) {
    var a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.textContent = "Download ready";
    a.download = name;
    return a;
  }

  // Start recording
  document.getElementById("startBtn").addEventListener("click", () => {
    if (mediaRecorder && mediaRecorder.state == "inactive") {
      mediaRecorder.start();
      toggleBtns(true, false, false);
    }
  });

  // Pause recording
  document.getElementById("pauseBtn").addEventListener("click", () => {
    if (mediaRecorder && (mediaRecorder.state !== "recording" || mediaRecorder.state !== "paused")) {
      $("#ipause").text(mediaRecorder.state == "paused" ? "resume" : "pause");
      mediaRecorder.state == "paused" ? mediaRecorder.resume() : mediaRecorder.pause();
      console.log("Recording paused/resumed");
    }
  });

  // Stop recording
  document.getElementById("stopBtn").addEventListener("click", () => {
    mediaRecorder?.stop();
    toggleBtns(false, true, true);
  });

  function postBlob(audioBlob, fileName, fileType) {
    audioBlob.arrayBuffer().then((buffer) => {
      const file = new File([buffer], fileName, {
        type: fileType,
      });
      const formData = new FormData();
      formData.append("file", file);
      fetch("http://localhost:8000/api/audio", {
        method: "POST",
        body: formData,
      })
        .then((response) => {
          console.log(response);
        })
        .catch((error) => {
          console.error(error);
        });
    });
  }
}
