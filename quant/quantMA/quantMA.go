package quantMA

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/paulfdunn/go-quantstudio/downloader"
	"github.com/paulfdunn/go-quantstudio/quant"
	"github.com/paulfdunn/logh"
)

type Group struct {
	Name   string
	Issues []Issue
}

type Issue struct {
	DownloaderIssue   *downloader.Issue
	QuantsetAsColumns QuantMA
}

type QuantMA struct {
	PriceNormalizedClose []float64
	PriceNormalizedHigh  []float64
	PriceNormalizedLow   []float64
	PriceNormalizedOpen  []float64
	PriceMA              []float64
	PriceMAHigh          []float64
	PriceMALow           []float64
	Results              quant.Results
}

var (
	appName string
)

func Init(appNameInit string) {
	appName = appNameInit
}

func GetGroup(downloaderGroup *downloader.Group, maLength int, maSplit float64) *Group {
	logh.Map[appName].Printf(logh.Info, "calling quant.Run with maLength: %d, maSplit: %5.2f", maLength, maSplit)
	group := Group{Name: downloaderGroup.Name}
	group.Issues = make([]Issue, len(downloaderGroup.Issues))

	for index := range downloaderGroup.Issues {
		// Dont use the looping variable in a "i,v" style for loop as
		// the variable is pointing to a pointer
		group.Issues[index] = Issue{DownloaderIssue: &downloaderGroup.Issues[index]}
		group.UpdateIssue(index, maLength, maSplit)
	}

	return &group
}

func (grp *Group) UpdateIssue(index int, maLength int, maSplit float64) {
	iss := grp.Issues[index].DownloaderIssue
	issDAC := iss.DatasetAsColumns
	priceNormalizedClose := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjClose)
	priceNormalizedHigh := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjHigh)
	priceNormalizedLow := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjLow)
	priceNormalizedOpen := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjOpen)
	priceMA := quant.MA(maLength, true, issDAC.AdjOpen, issDAC.AdjClose)
	priceMA = quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], priceMA)
	priceMALow := quant.MultiplySlice(1.0-maSplit, priceMA)
	priceMAHigh := quant.MultiplySlice(1.0+maSplit, priceMA)
	tradeMA := quant.TradeOnPrice(maLength, priceNormalizedClose, priceMA, priceMAHigh, priceMALow)
	tradeHistory, totalGain, tradeGainVsTime := quant.TradeGain(maLength, tradeMA, *iss)
	annualizedGain := quant.AnnualizedGain(totalGain, issDAC.Date[0], issDAC.Date[len(issDAC.Date)-1])
	results := quant.Results{AnnualizedGain: annualizedGain, TotalGain: totalGain, TradeHistory: tradeHistory,
		TradeMA: tradeMA, TradeGainVsTime: tradeGainVsTime}
	grp.Issues[index] = Issue{DownloaderIssue: iss,
		QuantsetAsColumns: QuantMA{PriceNormalizedClose: priceNormalizedClose,
			PriceNormalizedHigh: priceNormalizedHigh, PriceNormalizedLow: priceNormalizedLow,
			PriceNormalizedOpen: priceNormalizedOpen,
			PriceMA:             priceMA, PriceMAHigh: priceMAHigh, PriceMALow: priceMALow,
			Results: results}}
}

func WrappedPlotlyHandlerMA(dlGroupChan chan *downloader.Group) http.HandlerFunc {
	var dlGroup *downloader.Group
	var qGroup *Group
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		mal := r.URL.Query().Get("maLength")
		maLength, err := strconv.Atoi(mal)
		if err != nil {
			logh.Map[appName].Printf(logh.Error, "error converting maLength value '%s' to int", mal)
			return
		}
		mas := r.URL.Query().Get("maSplit")
		maSplit, err := strconv.ParseFloat(mas, 64)
		if err != nil {
			logh.Map[appName].Printf(logh.Error, "error converting maSplit value '%s' to float", mas)
			return
		}

		select {
		case dlGroup = <-dlGroupChan:
			qGroup = GetGroup(dlGroup, maLength, maSplit)
		default:
		}
		symbolIndex := -1
		for i, v := range dlGroup.Issues {
			if strings.EqualFold(v.Symbol, symbol) {
				symbolIndex = i
				break
			}
		}
		if symbolIndex == -1 {
			logh.Map[appName].Printf(logh.Warning, "Symbol %s was not found in existing data. Re-run with this symbol in the symbolCSVList.", symbol)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		qGroup.UpdateIssue(symbolIndex, maLength, maSplit)
		if err := plotlyJSON(qGroup.Issues[symbolIndex], w); err != nil {
			log.Printf("issue: %s", err)
		}
	}
}

// plotlyJSON writes plot data as JSON into w
func plotlyJSON(qIssue Issue, w io.Writer) error {
	reply := map[string]interface{}{
		// https://plotly.com/javascript/basic-charts/
		// https://plotly.com/javascript/reference/index/
		// color picker:
		// https://htmlcolorcodes.com/color-picker/
		"data": []map[string]interface{}{
			{
				"x":     qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"high":  qIssue.QuantsetAsColumns.PriceNormalizedHigh,
				"low":   qIssue.QuantsetAsColumns.PriceNormalizedLow,
				"open":  qIssue.QuantsetAsColumns.PriceNormalizedOpen,
				"close": qIssue.QuantsetAsColumns.PriceNormalizedClose,
				"name":  "Prices",
				"type":  "candlestick",
				// "xaxis": "x2",
			},
			{
				"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":    qIssue.QuantsetAsColumns.PriceMALow,
				"name": "MALow",
				"type": "scatter",
				// "stackgroup": "one",
				"line": map[string]interface{}{
					"color": "rgba(255,65,54,0.5)",
				},
				// "xaxis": "x2",
			},
			{
				"x":         qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":         qIssue.QuantsetAsColumns.PriceMA,
				"name":      "MA",
				"type":      "scatter",
				"fill":      "tonexty",
				"fillcolor": "rgba(255,164,157,0.3)",
				"line": map[string]interface{}{
					"color": "rgba(255,65,54,0.5)",
				},
				// "xaxis": "x2",
			},
			{
				"x":         qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":         qIssue.QuantsetAsColumns.PriceMAHigh,
				"name":      "MAHigh",
				"type":      "scatter",
				"fill":      "tonexty",
				"fillcolor": "rgba(0, 140, 8, 0.3)",
				"line": map[string]interface{}{
					"color": "rgba(0, 140, 8, 0.5)",
				},
				// "xaxis": "x2",
			},
			// Second chart
			{
				"x": qIssue.DownloaderIssue.DatasetAsColumns.Date,
				// "y":     qIssue.DownloaderIssue.DatasetAsColumns.Volume,
				// "name":  "Volume",
				"y":     qIssue.QuantsetAsColumns.Results.TradeMA,
				"name":  "TradeMA",
				"type":  "scatter",
				"yaxis": "y2",
				"line": map[string]interface{}{
					"color": "rgba(255, 172, 47,1.0)",
				},
			},
			{
				"x": qIssue.DownloaderIssue.DatasetAsColumns.Date,
				// "y":     qIssue.DownloaderIssue.DatasetAsColumns.Volume,
				// "name":  "Volume",
				"y":    qIssue.QuantsetAsColumns.Results.TradeGainVsTime,
				"name": "TradeGainVsTime",
				"type": "scatter",
				// "yaxis": "y3",
				"line": map[string]interface{}{
					"color": "rgba(0, 139, 147,1.0)",
				},
			},
		},
		"layout": map[string]interface{}{
			"spikedistance": 50,
			"hoverdistance": 50,
			"autosize":      true,
			"title":         qIssue.DownloaderIssue.Symbol,
			"grid": map[string]int{
				"rows":    1,
				"columns": 1,
			},
			"xaxis": map[string]interface{}{
				"domain": []float64{0.0, 0.9},
				// breaks vertical zoom
				// "fixedrange": true,
				"rangeslider": map[string]interface{}{
					"visible": false,
				},
				"showspikes":     true,
				"spikemode":      "across",
				"spikedash":      "solid",
				"spikecolor":     "#000000",
				"spikethickness": 1,
			},
			// Second chart
			// "xaxis2": map[string]interface{}{
			// breaks vertical zoom
			// 	"fixedrange": true,
			// 	"rangeslider": map[string]interface{}{
			// 		"visible": false,
			// 	},
			// 	"showspikes":     true,
			// 	"spikemode":      "across",
			// 	"spikedash":      "solid",
			// 	"spikecolor":     "#000000",
			// 	"spikethickness": 1,
			// },
			"yaxis": map[string]interface{}{
				"title":      "Price ($ - normalized), Trade Gain",
				"autorange":  true,
				"fixedrange": false,
				"type":       "log",
			},
			"yaxis2": map[string]interface{}{
				"title":     fmt.Sprintf("Trade (algorithm:moving average, buy=%d, sell=%d)", quant.Buy, quant.Sell),
				"autorange": false,
				"range":     []float64{0.0, 1.0},
				"tick0":     0,
				"dtick":     1.0,
				// "type":       "log",
				// Below are only needed when using single row
				"anchor":     "x",
				"overlaying": "y",
				"side":       "right",
			},
			// "yaxis3": map[string]interface{}{
			// 	"title":      "Trade Gain (%)",
			// 	"autorange":  true,
			// 	"fixedrange": false,
			// 	"type":       "log",
			// 	// Below are only needed when using single row
			// 	"anchor":     "free",
			// 	"overlaying": "y",
			// 	"side":       "right",
			// 	"position":   0.93,
			// },
		},
		"text": qIssue.QuantsetAsColumns.Results.TradeHistory,
	}

	return json.NewEncoder(w).Encode(reply)
}
