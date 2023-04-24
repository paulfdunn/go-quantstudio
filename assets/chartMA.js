async function updateChartMA() {
    let symbol = document.getElementById('symbol').value;
    let maLength = document.getElementById('maLength').value;
    let maSplit = document.getElementById('maSplit').value;
    let response = await fetch('/plotly-ma?symbol=' + symbol + '&maLength=' + maLength+ '&maSplit=' + maSplit);
    let reply = await response.json(); 
    Plotly.newPlot('chartMA', reply.data, reply.layout);
    tradeHistory.innerHTML = reply.text;
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('process').onclick = updateChartMA;
});
