async function toClipboard() {
  var val = document.getElementById("optimized").value;

  await navigator.clipboard.writeText(val);
}
