async function updateChartMA2() {
    let symbol = document.getElementById('symbol').value;
    let maLengthLF = document.getElementById('maLengthLF').value;
    let maLengthHF = document.getElementById('maLengthHF').value;
    let maShortShift = document.getElementById('maShortShift').value;
    let stopLoss = document.getElementById('stopLoss').value;
    let stopLossDelay = document.getElementById('stopLossDelay').value;
    let longQuickBuy = document.getElementById('longQuickBuy').checked;
    let ema = document.getElementById('ema').checked;
    let response = await fetch('/plotly-ma2?symbol=' + symbol + '&maLengthLF=' + maLengthLF+ '&maLengthHF=' + maLengthHF + '&maShortShift=' + maShortShift + '&stopLoss=' + stopLoss + '&stopLossDelay=' + stopLossDelay + '&longQuickBuy=' + longQuickBuy + '&ema=' + ema);
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
    document.getElementById('process').onclick = updateChartMA2;
    document.getElementById('ema').onclick = updateValues;
    document.getElementById('downloadData').onclick = downloadData;
});
