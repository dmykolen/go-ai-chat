var typeOpts = {
  strings: ["Lifecell", "Lifecell AI", "Lifecell STT", "Lifecell TTS"],
  typeSpeed: 150,
  loop: true,
  loopCount: Infinity,
  startDelay: 400,
  backSpeed: 40,
  smartBackspace: true,
  backDelay: 4000,
  showCursor: false,
  cursorChar: "|",
  autoInsertCss: true,
  bindInputFocusEvents: true,
  contentType: "html", // 'html' or 'null' for plaintext
};
var typed = null;

document.addEventListener("DOMContentLoaded", function () {
  typed = new Typed("#typeBrand", typeOpts);
  console.log("Typed: ", typed);
});
