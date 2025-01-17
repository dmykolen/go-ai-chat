// 2c0594a8-1289-4b82-bca2-9f0c6578f7b2
// 727c081a-2552-41b8-bd79-b6da5bacf23a
// 4d17ff36-b710-47d4-8442-172619b17999
// 3144f1ab-0dc1-49e2-ae91-e2567f108d58

function JSONIndent(json) {
  if (typeof json == 'string') {
      json = JSON.parse(json);
  }
  return JSON.stringify(json, null, 2);
}

function syntaxHighlight(json) {
  json = json.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  return json.replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, function (match) {
      var cls = 'text-pink-500';
      if (/^"/.test(match)) {
          cls = /:$/.test(match) ? 'text-red-500' : 'text-success';
      } else if (/true|false/.test(match)) {// 'boolean';
          cls = 'text-sky-500';
      } else if (/null/.test(match)) {// 'null';
          cls = 'text-purple-500'
      }
      return '<span class="' + cls + '">' + match + '</span>';
  });
}

function jsonStringPrepare(js) {
  if (!js || js.trim() === "") {
    return;
  }
  // Step 1: Trim and remove extra characters
  js = js.trim().replace(/\n `$/, "");

  // Step 2: Replace curly quotes with standard quotes
  js = js.replace(/[“”]/g, '"');
  return js;
}

function isValidJSON(value) {
  try {
    JSON.parse(value);
    return true;
  } catch (e) {
    console.warn("value is not JSON or invalid JSON! (" + value + ")");
    return false;
  }
}

function addTableToChatFromJSON(el, js) {
  if (!el) {
    console.error("Element is null or undefined.");
    return;
  }

  if ((!js || js.trim() === "") && el.text().trim() === "") {
    console.error("JSON is empty or null and element text is empty.");
    return;
  }

  const isJsonFromElement = !js;
  js = jsonStringPrepare(js || el.text());

  if (!isValidJSON(js)) {
    return;
  }

  let elTable = convertJSONtoHTMLTable(js);
  console.log("Table:", elTable);
  elTable.on("click", "tbody tr", function () {
    // let data = elTable.row(this).data();
    console.log("clicked on row", this);
  });

  // Check if elTable is not null and not empty
  if (elTable && elTable.html().trim() !== "") {
    isJsonFromElement && el.empty();
    el.html(elTable);
  } else {
    console.error("elTable is either null or empty.");
  }
  return elTable;
}

function convertJSONtoHTMLTable(js) {
  randomId = Math.floor(Math.random() * 1000);
  console.log("Random ID:", randomId);
  const tblTempl = `<table id="dt-${randomId}" class="table-sm display compact stripe nowrap"><thead></thead><tbody></tbody></table>`;
  const tableData = JSON.parse(js);

  // const formatString = tblTempl.replace('%s', 'dt-' + Math.floor(Math.random() * 1000));
  const columns = Object.keys(tableData[0]).map((key) => ({
    title: key,
    data: key,
  }));
  console.log("TableTemplate=%o\nCOLS: %o", tblTempl, columns);

  const elTable = $(tblTempl);
  elTable.DataTable({
    retrieve: true,
    searching: false,
    select: true,
    bAutoWidth: true,
    rowId: "CUSTOMER_CODE",
    data: tableData,
    columns: columns,
  });
  console.debug(elTable[0]);
  console.log("Add table with " + tableData.length + " rows and " + columns.length + " columns");
  return elTable;
}
