// @ts-nocheck
console.warn(">>>> ENV =>", process.env.NODE_ENV);

if (process.env.NODE_ENV !== "production") {
  import("daisyui/dist/full.css");
}
// import "../css/tailwind.css";

import "datatables.net-dt/css/dataTables.dataTables.min.css";
import "highlight.js/styles/base16/dracula.css";
// import "highlight.js/styles/github-dark.css";

import hljs from "highlight.js/lib/core";
import bash from "highlight.js/lib/languages/bash";
import go from "highlight.js/lib/languages/go";
import java from "highlight.js/lib/languages/java";
import javascript from "highlight.js/lib/languages/javascript";
import json from "highlight.js/lib/languages/json";
import plaintext from "highlight.js/lib/languages/plaintext";
import python from "highlight.js/lib/languages/python";
import shell from "highlight.js/lib/languages/shell";
import sql from "highlight.js/lib/languages/sql";
import xml from "highlight.js/lib/languages/xml";

import "datatables.net";
import htmx from "htmx.org";
import $ from "jquery";

import "marked";
import { markedHighlight } from "marked-highlight";
import { themeChange } from "theme-change";
import Typed from "typed.js";
themeChange()

// Create an object mapping language names to their modules
const languages = { go, sql, plaintext, javascript, java, python, bash, shell, json, xml };

// Loop over the object to register all languages
Object.entries(languages).forEach(([name, lang]) => {
  hljs.registerLanguage(name, lang);
});

// hljs.registerLanguage("go", go);
window.hljs = hljs;
window.$ = $;
window.markedHighlight = markedHighlight.markedHighlight;
window.htmx = htmx;
window.themeChange = themeChange;
window.Typed = Typed;

export function hlAllPreCode() {
  console.log("run hlAllPreCode");
  document.querySelectorAll("pre code:not(.hljs)").forEach((block) => {
    hljs.highlightElement(block);
  });
}
export function hlEl(el) {
  hljs.highlightElement(el);
}
export function hlGO(text) {
  return hljs.highlight(text, { language: "go" }).value;
}
export function hlSQL(text) {
  return hljs.highlight(text, { language: "sql" }).value;
}
export function hlJSON(text) {
  return hljs.highlight(text, { language: "json" }).value;
}
export function hlXML(text) {
  return hljs.highlight(text, { language: "xml" }).value;
}

window.hlAllPreCode = hlAllPreCode;
window.hlEl = hlEl;
window.hlGO = hlGO;
window.hlSQL = hlSQL;
window.hlJSON = hlJSON;
window.hlXML = hlXML;

import { Marked } from "marked";
const marked = new Marked(
  markedHighlight({
    langPrefix: "hljs language-",
    highlight(code, lang, info) {
      const language = hljs.getLanguage(lang) ? lang : "plaintext";
      return hljs.highlight(code, { language }).value;
    },
  })
);

if (marked && marked.version) {
  console.log(`marked library version ${marked.version} is loaded.`);
}
console.log(`TestMarked: %o`, marked.parse("# Marked in the browser\n\nRendered by **marked**.\n\n```js\nconsole.log('Hello, World!');\n```"));

export function mdParse(text) {
  return marked.parse(text.trim().replace(/ +/g, " ").replace(/\\n/g, "\n"));
}

window.marked = marked;
window.mdParse = mdParse;

// export default mdParse;

import Clipboard from "clipboard";
import "jquery-ui";
window.Clipboard = Clipboard;

export function addCopyBtnToCodeElems() {
  console.log("ADD copy button to code elements.");
  $("pre").has("code.hljs").addClass("relative");
  $("pre.relative").append(`<button name="btn-copy-code" class="absolute top-1 right-1 btn btn-xs btn-accent shadow-2xl shadow-black border-2 border-accent-content/20 w-8 delay-100 ease-in-out transition-all">copy</button>`);

  var cpp = new Clipboard('[name="btn-copy-code"]', {
      target: function (trigger) {
          console.log('TRIGGER <<Clipboard>>', trigger);
          console.log('TRIGGER <<Clipboard>> ParentElem:', trigger.parentElement);
          return trigger.parentElement;
      }
  });

  cpp.on('success', function (e) {
      console.info('Action:', e.action);
      console.info('Text:', e.text);
      console.info('Trigger:', e.trigger);

      e.clearSelection();
  });

  cpp.on('error', function (e) {
      console.error('Action:', e.action);
      console.error('Trigger:', e.trigger);
  });
}
window.addCopyBtnToCodeElems = addCopyBtnToCodeElems;

export { $, Clipboard, Typed };

