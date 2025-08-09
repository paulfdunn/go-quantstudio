async function downloadData() {
    let response = await fetch('/downloadData')

    if (!response.ok) {
        alert("downloadData did not successfully run and data WAS NOT loaded");
    } else {
        console.log("downloadData successful");
        alert("downloadData successfully ran");
    }
}

async function loadSymbols() {
    let response = await fetch('/symbols')
    let reply = await response.json();

    if (!response.ok) {
        alert("loadSymbols did not successfully load symbols");
    }

    symbols.innerHTML =  reply.join(" ");
}
