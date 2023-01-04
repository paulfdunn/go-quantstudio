async function updateChart() {
    let symbol = document.getElementById('symbol').value;
    let resp = await fetch('/plotly?symbol=' + symbol);
    let reply = await resp.json(); 
    Plotly.newPlot('chart', reply.data, reply.layout);
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('process').onclick = updateChart;
});
