package quantDir

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
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
	QuantsetAsColumns quantDir
}

type quantDir struct {
	PriceNormalizedClose []float64
	PriceNormalizedHigh  []float64
	PriceNormalizedLow   []float64
	PriceNormalizedOpen  []float64
	Direction            []float64
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

	// There are days where stocks largely move in one direction. On up days the low and open are of
	// similar value, as are close and high. On down days open/high are similar and close/low are similar.
	// See if those days mean anything.
	direction := make([]float64, len(priceNormalizedOpen))
	for i := 0; i < len(priceNormalizedOpen); i++ {
		// d1 := time.Date(2022, 5, 26, 7, 30, 0, 0, time.Local)
		// if iss.Symbol == "qqq" && issDAC.Date[i] == d1 {
		// 	fmt.Println("debug")
		// }
		endDiffUp := (priceNormalizedOpen[i] - priceNormalizedLow[i]) + (priceNormalizedHigh[i] - priceNormalizedClose[i])
		endDiffDown := (priceNormalizedHigh[i] - priceNormalizedOpen[i]) + (priceNormalizedClose[i] - priceNormalizedLow[i])
		smallEndDiff := math.Abs(endDiffUp/(priceNormalizedClose[i]-priceNormalizedOpen[i])) < 0.1 || math.Abs(endDiffDown/(priceNormalizedClose[i]-priceNormalizedOpen[i])) < 0.1
		if smallEndDiff &&
			math.Abs(priceNormalizedClose[i]-priceNormalizedOpen[i])/priceNormalizedOpen[i] > 0.01 {
			direction[i] += (priceNormalizedClose[i] - priceNormalizedOpen[i]) / priceNormalizedOpen[i]
		}
	}
	// direction = quant.MA(20, true, direction)
	// direction = quant.MultiplySlice(50, direction)
	direction = quant.OffsetSlice(0.5, direction)
	return Issue{DownloaderIssue: iss,
		QuantsetAsColumns: quantDir{PriceNormalizedClose: priceNormalizedClose,
			PriceNormalizedHigh: priceNormalizedHigh, PriceNormalizedLow: priceNormalizedLow,
			PriceNormalizedOpen: priceNormalizedOpen,
			Direction:           direction}}
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
			},
			// Second y-axis
			{
				"x":     qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":     qIssue.QuantsetAsColumns.Direction,
				"name":  "Direction",
				"type":  "scatter",
				"yaxis": "y2",
				"line": map[string]interface{}{
					"color": "rgba(47, 255, 61, 0.6)",
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
