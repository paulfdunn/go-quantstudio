async function updateChartDir() {
    let symbol = document.getElementById('symbol').value;
    let maLengthLF = document.getElementById('maLengthLF').value;
    let maLengthHF = document.getElementById('maLengthHF').value;
    let maShortShift = document.getElementById('maShortShift').value;
    let ema = document.getElementById('ema').checked;
    let response = await fetch('/plotly-ma2?symbol=' + symbol + '&maLengthLF=' + maLengthLF+ '&maLengthHF=' + maLengthHF + '&maShortShift=' + maShortShift + '&ema=' + ema);
    if (response.status >= 400 && response.status < 600) {
        Plotly.deleteTraces('chartMA2Chart', [0,1,2,3,4,5]);
        tradeHistory.innerHTML = "Server replied with error; likely an invalid symbol.";
        // throw new Error("Error response from server.");
        return;
    }
    let reply = await response.json();
    Plotly.newPlot('chartMA2Chart', reply.data, reply.layout);
    tradeHistory.innerHTML = reply.text;
}

async function updateValues() {
    if (document.getElementById('ema').checked) {
        document.getElementById('maLengthLF').value = 250;
        document.getElementById('maLengthHF').value = 80;
    } else {
        document.getElementById('maLengthLF').value = 150;
        document.getElementById('maLengthHF').value = 40;
    }
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('process').onclick = updateChartDir;
    document.getElementById('ema').onclick = updateValues;
});
