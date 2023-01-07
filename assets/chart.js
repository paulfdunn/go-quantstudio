async function updateChart() {
    let symbol = document.getElementById('symbol').value;
    let maLength = document.getElementById('maLength').value;
    let resp = await fetch('/plotly?symbol=' + symbol + '&maLength=' + maLength);
    let reply = await resp.json(); 
    Plotly.newPlot('chart', reply.data, reply.layout);
    trades.innerHTML = reply.text;
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('process').onclick = updateChart;
});
