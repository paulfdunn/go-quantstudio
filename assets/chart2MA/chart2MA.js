async function updateChartDir() {
    let symbol = document.getElementById('symbol').value;
    let maLengthLF = document.getElementById('maLengthLF').value;
    let maLengthHF = document.getElementById('maLengthHF').value;
    let response = await fetch('/plotly-mf?symbol=' + symbol + '&maLengthLF=' + maLengthLF+ '&maLengthHF=' + maLengthHF);
    if (response.status >= 400 && response.status < 600) {
        Plotly.deleteTraces('chart2MAChart', [0,1,2,3,4,5]);
        tradeHistory.innerHTML = "Server replied with error; likely an invalid symbol.";
        // throw new Error("Error response from server.");
        return;
    }
    let reply = await response.json();
    Plotly.newPlot('chart2MAChart', reply.data, reply.layout);
    tradeHistory.innerHTML = reply.text;
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('process').onclick = updateChartDir;
});
