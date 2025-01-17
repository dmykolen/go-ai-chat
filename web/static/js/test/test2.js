const escapedString = "Hello,\\nworld!\\n\\tTTTTTT";
console.log(escapedString);
const unescapedString = JSON.parse(`"${escapedString}"`);
console.log(unescapedString);

// const unescapedString2 = JSON.parse(escapedString);
// console.log(unescapedString2);
console.log("############################");
console.log(escapedString.replace(/\\n/g, "\n").replace(/\\t/g, "\t"));
console.log("############################");
console.log(escapedString.replace(/\\/g, "\\"));
console.log("############################");

// console.log("%c " + str, "color: green; font-weight: bold;");
function prettyLog(str) {
  console.log(str, "color: green; font-weight: bold;");
}

const jsonStr = `{
  "code": 200,
  "message": "OK",
  "data": [
    {
      "id": "ab482f19-6150-419d-9485-c5a3a039320b",
      "name": "",
      "createdTime": "2024-04-04T19:31:59.501",
      "lastUpdateTime": "1970-01-01T03:00:00.000"
    },
    {
      "id": "a7bbc185-92f9-45f7-9010-3196a8a12596",
      "name": "",
      "createdTime": "2024-04-04T19:31:59.503",
      "lastUpdateTime": "1970-01-01T03:00:00.000"
    },
    {
      "id": "c4310138-a502-4015-a252-dcf624f8eabe",
      "name": "",
      "createdTime": "2024-04-04T20:52:11.864",
      "lastUpdateTime": "1970-01-01T03:00:00.000"
    },
    {
      "id": "77644d56-0077-4428-a1c2-285ceae408e4",
      "name": "",
      "createdTime": "2024-04-04T20:52:26.441",
      "lastUpdateTime": "1970-01-01T03:00:00.000"
    },
    {
      "id": "fd4eb9f1-d62a-46a7-b8c4-774457b81c9a",
      "name": "",
      "createdTime": "2024-04-04T20:52:33.000",
      "lastUpdateTime": "1970-01-01T03:00:00.000"
    }
  ]
}`;

const jsonObj = JSON.parse(jsonStr);
console.log(jsonObj.data);

let resultHTML = "";
jsonObj.data.forEach((item) => {
  console.log("ArrItem ID=>", item.id, ";Name=>", item.name, ";CreatedTime=>", item.createdTime);
  // console.log(toHtml(item));
  resultHTML += toHtml(item);
});
console.log(resultHTML);

function toHtml(item) {
  return `
  <div class="flex flex-row space-x-2 mx-auto">
      <div class="flex justify-center pt-2 w-10 h-10 bg-primary rounded-full"><i class="fa-play fas scale-150 text-lg text-accent"></i></div>
      <div class="bg-base-200 rounded-lg p-2 relative">
          <div class="text-sm font-bold">${item.id}</div>
          <div class="absolute right-2 badge badge-xs badge-warning">${item.createdTime}</div>
      </div>
  </div>`;
}
