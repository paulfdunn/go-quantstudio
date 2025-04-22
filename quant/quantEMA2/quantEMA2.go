package quantEMA2

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/paulfdunn/go-helper/encodingh/v2/jsonh"
	"github.com/paulfdunn/go-helper/logh/v2"
	"github.com/paulfdunn/go-quantstudio/downloader"
	"github.com/paulfdunn/go-quantstudio/quant"
)

type Group struct {
	Name   string
	Issues []Issue
}

type Issue struct {
	DownloaderIssue   *downloader.Issue
	QuantsetAsColumns QuantEMA2
}

type QuantEMA2 struct {
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
	lp      func(level logh.LoghLevel, v ...interface{})
	lpf     func(level logh.LoghLevel, format string, v ...interface{})
)

func Init(appNameInit string) {
	appName = appNameInit
	lp = logh.Map[appName].Println
	lpf = logh.Map[appName].Printf
}

func (grp Group) String() string {
	out, err := json.MarshalIndent(grp, "", "  ")
	lpf(logh.Error, "calling json.MarshalIndent: %s", err)
	return string(jsonh.PrettyJSON(out))
}

func (iss Issue) String() string {
	out, err := json.MarshalIndent(iss, "", "  ")
	lpf(logh.Error, "calling json.MarshalIndent: %s", err)
	return string(jsonh.PrettyJSON(out))
}

func GetGroup(downloaderGroup *downloader.Group, tradingSymbols []string, maLengthLF int, maLengthHF int) *Group {
	lpf(logh.Info, "calling quant.Run with maLengthLF: %d, maLengthHF: %5.2f", maLengthLF, maLengthHF)
	group := Group{Name: downloaderGroup.Name}
	group.Issues = make([]Issue, len(downloaderGroup.Issues))

	for index := range downloaderGroup.Issues {
		// Skip non-trading symbols.
		if !slices.Contains(tradingSymbols, downloaderGroup.Issues[index].Symbol) {
			continue
		}
		// Dont use the looping variable in a "i,v" style for loop as
		// the variable is pointing to a pointer
		group.Issues[index] = Issue{DownloaderIssue: &downloaderGroup.Issues[index]}
		group.Issues[index] = UpdateIssue(group.Issues[index].DownloaderIssue, maLengthLF, maLengthHF)
	}

	return &group
}

func UpdateIssue(iss *downloader.Issue, maLengthLF int, maLengthHF int) Issue {
	issDAC := iss.DatasetAsColumns
	priceNormalizedClose := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLengthLF], issDAC.AdjClose)
	priceNormalizedHigh := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLengthLF], issDAC.AdjHigh)
	priceNormalizedLow := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLengthLF], issDAC.AdjLow)
	priceNormalizedOpen := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLengthLF], issDAC.AdjOpen)
	priceMALow := quant.EMA(maLengthLF, true, issDAC.AdjOpen, issDAC.AdjClose)
	priceMALow = quant.MultiplySlice(1.0/issDAC.AdjOpen[maLengthLF], priceMALow)
	priceMAHigh := quant.EMA(maLengthHF, true, issDAC.AdjOpen, issDAC.AdjClose)
	priceMAHigh = quant.MultiplySlice(1.0/issDAC.AdjOpen[maLengthLF], priceMAHigh)
	tradeMA := quant.TradeOnPrice(maLengthLF, priceMAHigh, priceMALow, priceMALow)
	tradeHistory, totalGain, tradeGainVsTime := quant.TradeGain(maLengthLF, tradeMA, *iss)
	annualizedGain := quant.AnnualizedGain(totalGain, issDAC.Date[0], issDAC.Date[len(issDAC.Date)-1])
	results := quant.Results{AnnualizedGain: annualizedGain, TotalGain: totalGain, TradeHistory: tradeHistory,
		Trade: tradeMA, TradeGainVsTime: tradeGainVsTime}
	return Issue{DownloaderIssue: iss,
		QuantsetAsColumns: QuantEMA2{PriceNormalizedClose: priceNormalizedClose,
			PriceNormalizedHigh: priceNormalizedHigh, PriceNormalizedLow: priceNormalizedLow,
			PriceNormalizedOpen: priceNormalizedOpen,
			PriceMAHigh:         priceMAHigh, PriceMALow: priceMALow,
			Results: results,
		}}
}

func WrappedPlotlyHandler(dlGroupChan chan *downloader.Group, tradingSymbols []string) http.HandlerFunc {
	// Use a closure to persist the state of these variables between calls. The first call to this
	// function must always have data in dlGroupChan. Subsequent calls will use the already
	// downloaded data and only process the single issue and parameters specified in the call. If
	// the user presses the Download button, new data IS downloaded and put in dlGroupChan, and that
	// data processed. This complexity allows fast processing without downloading data every call,
	// but also allows background downloading of (fresh) data and subsequent analysis.
	var dlGroup *downloader.Group
	var trdSymbols = tradingSymbols
	return func(w http.ResponseWriter, r *http.Request) {
		urlSymbol := strings.ToLower(r.URL.Query().Get("symbol"))
		malf := r.URL.Query().Get("maLengthLF")
		maLengthLF, err := strconv.Atoi(malf)
		if err != nil {
			lpf(logh.Error, "converting maLengthLF value '%s' to int", malf)
			return
		}
		mahf := r.URL.Query().Get("maLengthHF")
		maLengthHF, err := strconv.Atoi(mahf)
		if err != nil {
			lpf(logh.Error, "converting maLengthHF value '%s' to int", mahf)
			return
		}

		select {
		case dlGroup = <-dlGroupChan:
		default:
			lp(logh.Debug, "using previously downloaded data")
		}
		if !slices.Contains(trdSymbols, urlSymbol) {
			lpf(logh.Warning, "Symbol %s was not found in existing data. Re-run with this symbol in the symbolCSVList.", urlSymbol)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		symbolIndex := -1
		for i, v := range dlGroup.Issues {
			if strings.EqualFold(v.Symbol, urlSymbol) {
				symbolIndex = i
				break
			}
		}
		if symbolIndex == -1 {
			lpf(logh.Warning, "Symbol %s was not found in existing data. Re-run with this symbol in the symbolCSVList.", urlSymbol)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		iss := UpdateIssue(&dlGroup.Issues[symbolIndex], maLengthLF, maLengthHF)
		if err := plotlyJSON(iss, w); err != nil {
			lpf(logh.Error, "issue: %s\n%+v", err, iss)
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
			},
			{
				"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":    qIssue.QuantsetAsColumns.PriceMAHigh,
				"name": "MAHigh",
				"type": "scatter",
				"line": map[string]interface{}{
					"color": "rgba(0, 140, 8, 0.5)",
				},
			},
			{
				"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":    qIssue.QuantsetAsColumns.Results.TradeGainVsTime,
				"name": "TradeGainVsTime",
				"type": "scatter",
				// "yaxis": "y3",
				"line": map[string]interface{}{
					"color": "rgba(0, 139, 147, 1)",
				},
			},
			// Second y-axis
			{
				"x":     qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":     qIssue.QuantsetAsColumns.Results.Trade,
				"name":  "Trade",
				"type":  "scatter",
				"yaxis": "y2",
				"line": map[string]interface{}{
					"color": "rgba(255, 172, 47, 1)",
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
				"title":          "Price ($ - normalized), Trade Gain",
				"autorange":      true,
				"fixedrange":     false,
				"showspikes":     true,
				"spikemode":      "across",
				"spikedash":      "solid",
				"spikecolor":     "#000000",
				"spikethickness": 1,
				"type":           "log",
			},
			"yaxis2": map[string]interface{}{
				"title":          fmt.Sprintf("Trade (algorithm:moving average, buy=%d, sell=%d)", quant.Buy, quant.Sell),
				"autorange":      false,
				"range":          []float64{0.0, 1.0},
				"showspikes":     true,
				"spikemode":      "across",
				"spikedash":      "solid",
				"spikecolor":     "#000000",
				"spikethickness": 1,
				"tick0":          0,
				"dtick":          1.0,
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
