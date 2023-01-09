async function updateChart() {
    let symbol = document.getElementById('symbol').value;
    let maLength = document.getElementById('maLength').value;
    let response = await fetch('/plotly?symbol=' + symbol + '&maLength=' + maLength);
    let reply = await response.json(); 
    Plotly.newPlot('chart', reply.data, reply.layout);
    tradeHistory.innerHTML = reply.text;
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('process').onclick = updateChart;
});

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

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('downloadData').onclick = downloadData;
});