async function copyText() {
  var copyText = document.getElementById("prompt");

  try {
    const permissionResult = await navigator.permissions.query({
      name: "clipboard-write",
    });

    if (
      permissionResult.state === "granted" ||
      permissionResult.state === "prompt"
    ) {
      // Only works under HTTPS
      await navigator.clipboard.writeText(copyText.value);
    } else {
      console.error("Clipboard write permission denied");
    }
  } catch (err) {
    console.error("Error in copying text: ", err);
  }
}
