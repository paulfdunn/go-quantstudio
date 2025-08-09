async function updateChartMAH() {
    let symbol = document.getElementById('symbol').value;
    let maLength = document.getElementById('maLength').value;
    let maSplit = document.getElementById('maSplit').value;
    let maShortShift = document.getElementById('maShortShift').value;
    let stopLoss = document.getElementById('stopLoss').value;
    let stopLossDelay = document.getElementById('stopLossDelay').value;
    let longQuickBuy = document.getElementById('longQuickBuy').checked;
    let ema = document.getElementById('ema').checked;
    let response = await fetch('/plotly-mah?symbol=' + symbol + '&maLength=' + maLength+ '&maSplit=' + maSplit+ '&maShortShift=' + maShortShift + '&stopLoss=' + stopLoss + '&stopLossDelay=' + stopLossDelay + '&longQuickBuy=' + longQuickBuy + '&ema=' + ema);
    if (response.status >= 400 && response.status < 600) {
        Plotly.deleteTraces('chartMAHChart', [0,1,2,3,4,5]);
        tradeHistory.innerHTML = "Server replied with error; likely an invalid symbol.";
        // throw new Error("Error response from server.");
        return;
    }
    let reply = await response.json();
    Plotly.newPlot('chartMAHChart', reply.data, reply.layout);
    tradeHistory.innerHTML = reply.text;
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('process').onclick = updateChartMAH;
    document.getElementById('downloadData').onclick = downloadData;
});
