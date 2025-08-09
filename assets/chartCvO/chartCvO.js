async function updateChartCvO() {
    let symbol = document.getElementById('symbol').value;
    let maLength = document.getElementById('maLength').value;
    let maSplit = document.getElementById('maSplit').value;
    let response = await fetch('/plotly-cvo?symbol=' + symbol + '&maLength=' + maLength+ '&maSplit=' + maSplit);
    if (response.status >= 400 && response.status < 600) {
        Plotly.deleteTraces('chartCvOChart', [0,1,2,3,4,5]);
        tradeHistory.innerHTML = "Server replied with error; likely an invalid symbol.";
        // throw new Error("Error response from server.");
        return;
    }
    let reply = await response.json();
    Plotly.newPlot('chartCvOChart', reply.data, reply.layout);
    tradeHistory.innerHTML = reply.text;
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('process').onclick = updateChartCvO;
    document.getElementById('downloadData').onclick = downloadData;
});
