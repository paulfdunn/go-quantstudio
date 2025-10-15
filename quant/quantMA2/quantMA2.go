package quantMA2

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
	QuantsetAsColumns QuantMA2
}

type QuantMA2 struct {
	Direction []int
	// DirectionMA          []float64
	PriceNormalizedClose []float64
	PriceNormalizedHigh  []float64
	PriceNormalizedLow   []float64
	PriceNormalizedOpen  []float64
	PriceMA              []float64
	PriceMAHigh          []float64
	PriceMALow           []float64
	PriceMALowShort      []float64
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

func GetGroup(downloaderGroup *downloader.Group, tradingSymbols []string, maLengthLF int, maLengthHF int, maShortShift float64, stopLoss float64, stopLossDelay int, longQuickBuyChecked, emaChecked bool) *Group {
	lpf(logh.Info, "calling quant.Run with maLengthLF: %d, maLengthHF: %d, maShortShift: %5.2", maLengthLF, maLengthHF, maShortShift)
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
		group.Issues[index] = UpdateIssue(group.Issues[index].DownloaderIssue, maLengthLF, maLengthHF, maShortShift, stopLoss, stopLossDelay, longQuickBuyChecked, emaChecked)
	}

	return &group
}

func UpdateIssue(iss *downloader.Issue, maLengthLF int, maLengthHF int, maShortShift float64, stopLoss float64, stopLossDelay int, longQuickBuyChecked, emaChecked bool) Issue {
	issDAC := iss.DatasetAsColumns
	priceNormalizedClose := quant.MultiplySlice(1.0/issDAC.Open[maLengthLF], issDAC.Close)
	priceNormalizedHigh := quant.MultiplySlice(1.0/issDAC.Open[maLengthLF], issDAC.High)
	priceNormalizedLow := quant.MultiplySlice(1.0/issDAC.Open[maLengthLF], issDAC.Low)
	priceNormalizedOpen := quant.MultiplySlice(1.0/issDAC.Open[maLengthLF], issDAC.Open)
	var priceMALf []float64
	var err error
	if emaChecked {
		priceMALf, err = quant.EMA(maLengthLF, true, priceNormalizedOpen, priceNormalizedClose)
	} else {
		priceMALf, err = quant.MA(maLengthLF, true, priceNormalizedOpen, priceNormalizedClose)
	}
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	var priceMAHf []float64
	if emaChecked {
		priceMAHf, err = quant.EMA(maLengthHF, true, priceNormalizedOpen, priceNormalizedClose)
	} else {
		priceMAHf, err = quant.MA(maLengthHF, true, priceNormalizedOpen, priceNormalizedClose)
	}
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	// Shifting the low frequency moving average down for short trades makes those
	// trades "harder" to enter and provides separation between the long and short trades.
	shortMA := quant.MultiplySlice(maShortShift, priceMALf)
	var rebuy quant.TradeOnSignalLongQuickBuyInputs
	if longQuickBuyChecked {
		rebuy = quant.TradeOnSignalLongQuickBuyInputs{DlIssue: iss, AllowedLongQuickBuys: 3, ConsecutiveUpDays: 7, Stop: 0.95}
	}
	tradeMA, err := quant.TradeOnSignal(&rebuy, maLengthLF, priceMAHf, priceMALf, priceMALf, shortMA, shortMA)
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	tradeMA = quant.TradeAddStop(tradeMA, stopLoss, stopLossDelay, *iss)
	dir := quant.ConsecutiveDirection(iss.DatasetAsColumns.Close)
	// dirMA, _ := quant.MA(10, true, quant.IntSliceToFloatSlice(quant.Direction(iss.DatasetAsColumns.Close)))
	tradeHistory, totalGain, tradeGainVsTime := quant.TradeGain(maLengthLF, tradeMA, *iss)
	annualizedGain := quant.AnnualizedGain(totalGain, issDAC.Date[0], issDAC.Date[len(issDAC.Date)-1])
	results := quant.Results{AnnualizedGain: annualizedGain, TotalGain: totalGain, TradeHistory: tradeHistory,
		Trade: tradeMA, TradeGainVsTime: tradeGainVsTime}
	return Issue{DownloaderIssue: iss,
		QuantsetAsColumns: QuantMA2{
			Direction: dir,
			// DirectionMA:          dirMA,
			PriceNormalizedClose: priceNormalizedClose,
			PriceNormalizedHigh:  priceNormalizedHigh, PriceNormalizedLow: priceNormalizedLow,
			PriceNormalizedOpen: priceNormalizedOpen,
			PriceMAHigh:         priceMAHf, PriceMALow: priceMALf, PriceMALowShort: shortMA,
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
		mass := r.URL.Query().Get("maShortShift")
		maShortShift, err := strconv.ParseFloat(mass, 64)
		if err != nil {
			lpf(logh.Error, "converting maShortShift value '%s' to float", mass)
			return
		}
		sl := r.URL.Query().Get("stopLoss")
		stopLoss, err := strconv.ParseFloat(sl, 64)
		if err != nil {
			lpf(logh.Error, "converting stopLoss value '%s' to float", sl)
			return
		}
		sld := r.URL.Query().Get("stopLossDelay")
		stopLossDelay, err := strconv.Atoi(sld)
		if err != nil {
			lpf(logh.Error, "converting stopLossDelay value '%s' to int", sld)
			return
		}
		longQuickBuy := r.URL.Query().Get("longQuickBuy")
		longQuickBuyChecked := false
		if strings.EqualFold(longQuickBuy, "true") {
			longQuickBuyChecked = true
		}
		ema := r.URL.Query().Get("ema")
		emaChecked := false
		if strings.EqualFold(ema, "true") {
			emaChecked = true
		}
		select {
		case dlGroup = <-dlGroupChan:
		default:
			lp(logh.Debug, "using previously downloaded data")
		}
		if !slices.Contains(trdSymbols, urlSymbol) {
			lpf(logh.Warning, "Symbol %s was not found in the symbolCSVList.", urlSymbol)
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
			lpf(logh.Warning, "Symbol %s is in the symbolCSVList but there is no matching Issue in dlGroup.Issues.", urlSymbol)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		iss := UpdateIssue(&dlGroup.Issues[symbolIndex], maLengthLF, maLengthHF, maShortShift, stopLoss, stopLossDelay, longQuickBuyChecked, emaChecked)
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
				"y":    qIssue.QuantsetAsColumns.PriceMALowShort,
				"name": "MALowShort",
				"type": "scatter",
				// "stackgroup": "one",
				"line": map[string]interface{}{
					"color": "rgba(255, 64, 54, 0.19)",
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
			// Third y-axis
			{
				"x":     qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":     qIssue.QuantsetAsColumns.Direction,
				"name":  "Direction",
				"type":  "scatter",
				"yaxis": "y3",
				"line": map[string]interface{}{
					"color": "rgba(206, 204, 199, 0.3)",
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
				"title":          fmt.Sprintf("Trade (algorithm:moving average, longBuy=%d, close=%d)", quant.LongBuy, quant.Close),
				"autorange":      false,
				"range":          []float64{-1.0, 2.0},
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
			"yaxis3": map[string]interface{}{
				"title":          "Direction",
				"autorange":      false,
				"range":          []float64{-10.0, 10.0},
				"showspikes":     true,
				"spikemode":      "across",
				"spikedash":      "solid",
				"spikecolor":     "#000000",
				"spikethickness": 1,
				"tick0":          0,
				"dtick":          1.0,
				// "type":       "log",
				// Below are only needed when using single row
				"anchor":     "free",
				"overlaying": "y",
				"side":       "right",
				"position":   1.15,
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
