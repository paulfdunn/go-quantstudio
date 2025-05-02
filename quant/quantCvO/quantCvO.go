package quantCvO

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
	QuantsetAsColumns QuantCvO
}

type QuantCvO struct {
	PriceNormalizedClose   []float64
	PriceNormalizedHigh    []float64
	PriceNormalizedLow     []float64
	PriceNormalizedOpen    []float64
	PriceMA                []float64
	PriceMAHigh            []float64
	PriceMALow             []float64
	GainMarketClosed       []float64
	GainMarketClosedMA     []float64
	GainMarketClosedMAHigh []float64
	GainMarketClosedMALow  []float64
	GainMarketOpen         []float64
	GainMarketOpenMA       []float64
	GainMarketOpenMAHigh   []float64
	GainMarketOpenMALow    []float64
	SlopeC                 []float64
	SlopeO                 []float64
	SlopeCvO               []float64
	Results                quant.Results
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

func GetGroup(downloaderGroup *downloader.Group, tradingSymbols []string, maLength int, maSplit float64) *Group {
	lpf(logh.Info, "calling quant.Run with maLength: %d, maSplit: %5.2f", maLength, maSplit)
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
		group.Issues[index] = UpdateIssue(group.Issues[index].DownloaderIssue, maLength, maSplit)
	}

	return &group
}

func UpdateIssue(iss *downloader.Issue, maLength int, maSplit float64) Issue {
	issDAC := iss.DatasetAsColumns
	priceNormalizedClose := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjClose)
	priceNormalizedHigh := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjHigh)
	priceNormalizedLow := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjLow)
	priceNormalizedOpen := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjOpen)
	gainMarketClosed, err := quant.MarketClosedGain(issDAC.AdjOpen, issDAC.AdjClose)
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	gainNormalizedMarketClosed := quant.MultiplySlice(1.0/gainMarketClosed[maLength], gainMarketClosed)
	gainMarketClosedMA, err := quant.EMA(maLength, true, gainNormalizedMarketClosed)
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	gainMarketClosedMALow := quant.MultiplySlice(1.0-maSplit, gainMarketClosedMA)
	gainMarketClosedMAHigh := quant.MultiplySlice(1.0+maSplit, gainMarketClosedMA)
	gainMarketOpen, err := quant.MarketOpenGain(issDAC.AdjOpen, issDAC.AdjClose)
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	gainNormalizedMarketOpen := quant.MultiplySlice(1.0/gainMarketOpen[maLength], gainMarketOpen)
	gainMarketOpenMA, err := quant.EMA(maLength, true, gainNormalizedMarketOpen)
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	gainMarketOpenMALow := quant.MultiplySlice(1.0-maSplit, gainMarketOpenMA)
	gainMarketOpenMAHigh := quant.MultiplySlice(1.0+maSplit, gainMarketOpenMA)

	slopeC, err := quant.MultiplySlices(quant.Differentiate(gainMarketClosedMA), quant.ReciprocolSlice(gainMarketClosedMA))
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	slopeC, err = quant.EMA(30, false, slopeC)
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	slopeC = quant.MultiplySlice(200, slopeC)

	slopeO, err := quant.MultiplySlices(quant.Differentiate(gainMarketOpenMA), quant.ReciprocolSlice(gainMarketOpenMA))
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	slopeO, err = quant.EMA(30, false, slopeO)
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	slopeO = quant.MultiplySlice(200, slopeO)
	slopeCvO, err := quant.SumSlices(slopeC, slopeO)
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}
	tradeLevel := make([]float64, len(slopeCvO))

	tradeCvO, err := quant.TradeOnSignal(maLength, slopeCvO,
		quant.OffsetSlice(maSplit, tradeLevel),
		quant.OffsetSlice(-maSplit, tradeLevel),
		nil,
		nil)
	if err != nil {
		lpf(logh.Error, "symbol: %s, %+v", iss.Symbol, err)
		return Issue{}
	}

	tradeHistory, totalGain, tradeGainVsTime := quant.TradeGain(maLength, tradeCvO, *iss)
	annualizedGain := quant.AnnualizedGain(totalGain, issDAC.Date[0], issDAC.Date[len(issDAC.Date)-1])
	results := quant.Results{AnnualizedGain: annualizedGain,
		TotalGain: totalGain, TradeHistory: tradeHistory,
		Trade: tradeCvO, TradeGainVsTime: tradeGainVsTime}
	return Issue{DownloaderIssue: iss,
		QuantsetAsColumns: QuantCvO{PriceNormalizedClose: priceNormalizedClose,
			PriceNormalizedHigh: priceNormalizedHigh, PriceNormalizedLow: priceNormalizedLow,
			PriceNormalizedOpen: priceNormalizedOpen,
			GainMarketClosed:    gainNormalizedMarketClosed, GainMarketClosedMA: gainMarketClosedMA,
			GainMarketClosedMAHigh: gainMarketClosedMAHigh, GainMarketClosedMALow: gainMarketClosedMALow,
			GainMarketOpen: gainNormalizedMarketOpen, GainMarketOpenMA: gainMarketOpenMA,
			GainMarketOpenMAHigh: gainMarketOpenMAHigh, GainMarketOpenMALow: gainMarketOpenMALow,
			SlopeC: slopeC, SlopeO: slopeO, SlopeCvO: slopeCvO,
			Results: results}}
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
		mal := r.URL.Query().Get("maLength")
		maLength, err := strconv.Atoi(mal)
		if err != nil {
			lpf(logh.Error, "converting maLength value '%s' to int", mal)
			return
		}
		mas := r.URL.Query().Get("maSplit")
		maSplit, err := strconv.ParseFloat(mas, 64)
		if err != nil {
			lpf(logh.Error, "converting maSplit value '%s' to float", mas)
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
		iss := UpdateIssue(&dlGroup.Issues[symbolIndex], maLength, maSplit)
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
				// "xaxis": "x2",
			},
			// {
			// 	"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
			// 	"y":    qIssue.QuantsetAsColumns.PriceMALow,
			// 	"name": "MALow",
			// 	"type": "scatter",
			// 	// "stackgroup": "one",
			// 	"line": map[string]interface{}{
			// 		"color": "rgba(255,65,54,0.5)",
			// 	},
			// },
			// {
			// 	"x":         qIssue.DownloaderIssue.DatasetAsColumns.Date,
			// 	"y":         qIssue.QuantsetAsColumns.PriceMA,
			// 	"name":      "MA",
			// 	"type":      "scatter",
			// 	"fill":      "tonexty",
			// 	"fillcolor": "rgba(255,164,157,0.3)",
			// 	"line": map[string]interface{}{
			// 		"color": "rgba(255,65,54,0.5)",
			// 	},
			// },
			// {
			// 	"x":         qIssue.DownloaderIssue.DatasetAsColumns.Date,
			// 	"y":         qIssue.QuantsetAsColumns.PriceMAHigh,
			// 	"name":      "MAHigh",
			// 	"type":      "scatter",
			// 	"fill":      "tonexty",
			// 	"fillcolor": "rgba(0, 140, 8, 0.3)",
			// 	"line": map[string]interface{}{
			// 		"color": "rgba(0, 140, 8, 0.5)",
			// 	},
			// },
			// {
			// 	"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
			// 	"y":    qIssue.QuantsetAsColumns.GainMarketClosedMALow,
			// 	"name": "GainMarketClosedMALow",
			// 	"type": "scatter",
			// 	// "stackgroup": "one",
			// 	"line": map[string]interface{}{
			// 		"color": "rgba(255,65,54,0.5)",
			// 	},
			// },
			{
				"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":    qIssue.QuantsetAsColumns.GainMarketClosedMA,
				"name": "GainMarketClosedMA",
				"type": "scatter",
				// "fill":      "tonexty",
				// "fillcolor": "rgba(255,164,157,0.3)",
				"line": map[string]interface{}{
					"color": "rgba(255,65,54,0.5)",
				},
			},
			// {
			// 	"x":         qIssue.DownloaderIssue.DatasetAsColumns.Date,
			// 	"y":         qIssue.QuantsetAsColumns.GainMarketClosedMAHigh,
			// 	"name":      "GainMarketClosedMAHigh",
			// 	"type":      "scatter",
			// 	"fill":      "tonexty",
			// 	"fillcolor": "rgba(0, 140, 8, 0.3)",
			// 	"line": map[string]interface{}{
			// 		"color": "rgba(0, 140, 8, 0.5)",
			// 	},
			// },
			{
				"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":    qIssue.QuantsetAsColumns.GainMarketClosed,
				"name": "GainMarketClosed",
				"type": "scatter",
				"line": map[string]interface{}{
					"color": "rgba(255, 0, 0,1)",
				},
			},
			// {
			// 	"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
			// 	"y":    qIssue.QuantsetAsColumns.GainMarketOpenMALow,
			// 	"name": "GainMarketOpenMALow",
			// 	"type": "scatter",
			// 	// "stackgroup": "one",
			// 	"line": map[string]interface{}{
			// 		"color": "rgba(255,65,54,0.5)",
			// 	},
			// },
			{
				"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":    qIssue.QuantsetAsColumns.GainMarketOpenMA,
				"name": "GainMarketOpenMA",
				"type": "scatter",
				// "fill":      "tonexty",
				// "fillcolor": "rgba(255,164,157,0.3)",
				"line": map[string]interface{}{
					"color": "rgba(36, 166, 41, 0.39)",
				},
			},
			// {
			// 	"x":         qIssue.DownloaderIssue.DatasetAsColumns.Date,
			// 	"y":         qIssue.QuantsetAsColumns.GainMarketOpenMAHigh,
			// 	"name":      "GainMarketOpenMAHigh",
			// 	"type":      "scatter",
			// 	"fill":      "tonexty",
			// 	"fillcolor": "rgba(0, 140, 8, 0.3)",
			// 	"line": map[string]interface{}{
			// 		"color": "rgba(0, 140, 8, 0.5)",
			// 	},
			// },
			{
				"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":    qIssue.QuantsetAsColumns.GainMarketOpen,
				"name": "GainMarketOpen",
				"type": "scatter",
				"line": map[string]interface{}{
					"color": "rgba(0, 255, 0,1)",
				},
			},
			{
				"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":    qIssue.QuantsetAsColumns.Results.TradeGainVsTime,
				"name": "TradeGainVsTime",
				"type": "scatter",
				"line": map[string]interface{}{
					"color": "rgba(0, 139, 147,1)",
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
					"color": "rgba(255, 172, 47,1)",
				},
			},
			{
				"x":     qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":     qIssue.QuantsetAsColumns.SlopeCvO,
				"name":  "SlopeCvO",
				"type":  "scatter",
				"yaxis": "y2",
				"line": map[string]interface{}{
					"color": "rgba(255, 47, 172, 1)",
				},
			},
			{
				"x":     qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":     qIssue.QuantsetAsColumns.SlopeC,
				"name":  "SlopeC",
				"type":  "scatter",
				"yaxis": "y2",
				"line": map[string]interface{}{
					"color": "rgba(5, 69, 233, 0.43)",
				},
			},
			{
				"x":     qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":     qIssue.QuantsetAsColumns.SlopeO,
				"name":  "SlopeO",
				"type":  "scatter",
				"yaxis": "y2",
				"line": map[string]interface{}{
					"color": "rgba(5, 189, 57, 0.56)",
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
				"range":          []float64{-1.0, 1.0},
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
		},
		"text": qIssue.QuantsetAsColumns.Results.TradeHistory,
	}

	return json.NewEncoder(w).Encode(reply)
}
