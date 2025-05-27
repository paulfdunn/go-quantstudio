// Package financeYahooChart implements a price downloader using the finance.yahoo chart API.
// This bypasses the paywall.
package financeYahooChart

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/paulfdunn/go-helper/logh/v2"
	"github.com/paulfdunn/go-helper/mathh/v2"
	"github.com/paulfdunn/go-helper/neth/v2/httph"
	dl "github.com/paulfdunn/go-quantstudio/downloader"
)

type YfChartObj struct {
	Chart YfChart `json:"chart"`
}

type YfChart struct {
	Result []YfResult `json:"result"`
}

type YfResult struct {
	Meta       struct{} `json:"meta"`
	Timestamp  []int    `json:"timestamp"`
	Indicators struct {
		Quote    []YfQuote    `json:"quote"`
		AdjClose []YfAdjClose `json:"adjclose"`
	} `json:"indicators"`
}

type YfQuote struct {
	Close  []float64 `json:"close"`
	High   []float64 `json:"high"`
	Low    []float64 `json:"low"`
	Open   []float64 `json:"open"`
	Volume []int     `json:"volume"`
}

type YfAdjClose struct {
	AdjClose []float64 `json:"adjclose"`
}

var (
	appName string
	// lp      func(level logh.LoghLevel, v ...interface{})
	lpf func(level logh.LoghLevel, format string, v ...interface{})

	// See yfinance for parameter reference:
	// https://github.com/ranaroussi/yfinance/blob/3fe87cb1326249cb6a2ce33e9e23c5fd564cf54b/yfinance/scrapers/history.py#L13
	yahooURL = "https://query2.finance.yahoo.com/v8/finance/chart/%s?period1=%d&period2=%d&" +
		"interval=1d"
)

func Init(appNameInit string) {
	appName = appNameInit
	// lp = logh.Map[appName].Println
	lpf = logh.Map[appName].Printf
}

func NewGroup(liveData bool, dataFilePath string, name string, symbols []string) (*dl.Group, error) {
	return dl.NewGroup(liveData, dataFilePath, name, symbols, yahooURL, urlCollectionDataToGroup)
}

// urlCollectionDataToGroup processes raw data into a Group
func urlCollectionDataToGroup(urlData []httph.URLCollectionData, urlSymbolMap map[string]string, name string) (group *dl.Group, err error) {
	group = new(dl.Group)
	group.Name = name
	for _, ucd := range urlData {
		symbol, ok := urlSymbolMap[dl.BaseURL(ucd.URL)]
		if !ok {
			lpf(logh.Error, "the URL is not in map, URL: %s", dl.BaseURL(ucd.URL))
		}
		ts := make([]int, 0)
		yfr := make([]YfResult, 1)
		yfr[0].Timestamp = ts
		yfc := YfChartObj{Chart: YfChart{Result: yfr}}
		err := json.Unmarshal(ucd.Bytes, &yfc)
		if err != nil {
			lpf(logh.Error, "unmarshal of data failed, symbol: %s, body:%s, error:%s", symbol, string(ucd.Bytes), err)
			return nil, err
		}
		if yfc.Chart.Result == nil {
			lpf(logh.Error, "zero length data, symbol:%5s, body:%s", symbol, string(ucd.Bytes))
			return nil, errors.New("zero length data")
		}

		yfcLen := len(yfc.Chart.Result[0].Timestamp)
		Date := make([]time.Time, 0, yfcLen)
		Open := make([]float64, 0, yfcLen)
		High := make([]float64, 0, yfcLen)
		Low := make([]float64, 0, yfcLen)
		Close := make([]float64, 0, yfcLen)
		Volume := make([]float64, 0, yfcLen)
		AdjOpen := make([]float64, 0, yfcLen)
		AdjHigh := make([]float64, 0, yfcLen)
		AdjLow := make([]float64, 0, yfcLen)
		AdjClose := make([]float64, 0, yfcLen)
		AdjVolume := make([]float64, 0, yfcLen)
		datasetAsColumns := dl.DatasetAsColumns{
			Date: Date, Open: Open, High: High, Low: Low, Close: Close, Volume: Volume,
			AdjOpen: AdjOpen, AdjHigh: AdjHigh, AdjLow: AdjLow, AdjClose: AdjClose, AdjVolume: AdjVolume}
		dateFirst := time.Unix(int64(yfc.Chart.Result[0].Timestamp[0]), 0)
		dateLast := time.Unix(int64(yfc.Chart.Result[0].Timestamp[yfcLen-1]), 0)
		for i := 0; i < yfcLen; i++ {
			date := time.Unix(int64(yfc.Chart.Result[0].Timestamp[i]), 0)
			if yfc.Chart.Result[0].Indicators.Quote[0].Open[i] == 0 {
				// Some issues have weekend and holiday dates with price data that is all zeros.
				if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
					continue
				}
				// Print is noisy for ^tnx - only print for debugging.
				// lpf(logh.Warning, "Open is zero on weekday (may be holiday), symbol: %s, date: %s", symbol, date)
				continue
			}
			datasetAsColumns.Date = append(datasetAsColumns.Date, time.Unix(int64(yfc.Chart.Result[0].Timestamp[i]), 0))
			datasetAsColumns.Open = append(datasetAsColumns.Open, yfc.Chart.Result[0].Indicators.Quote[0].Open[i])
			datasetAsColumns.High = append(datasetAsColumns.High, yfc.Chart.Result[0].Indicators.Quote[0].High[i])
			datasetAsColumns.Low = append(datasetAsColumns.Low, yfc.Chart.Result[0].Indicators.Quote[0].Low[i])
			datasetAsColumns.Close = append(datasetAsColumns.Close, yfc.Chart.Result[0].Indicators.Quote[0].Close[i])
			datasetAsColumns.Volume = append(datasetAsColumns.Volume, float64(yfc.Chart.Result[0].Indicators.Quote[0].Volume[i]))
			adj := yfc.Chart.Result[0].Indicators.AdjClose[0].AdjClose[i] / yfc.Chart.Result[0].Indicators.Quote[0].Close[i]
			datasetAsColumns.AdjOpen = append(datasetAsColumns.AdjOpen, mathh.Round(yfc.Chart.Result[0].Indicators.Quote[0].Open[i]*adj, dl.InputPrecision))
			datasetAsColumns.AdjHigh = append(datasetAsColumns.AdjHigh, mathh.Round(yfc.Chart.Result[0].Indicators.Quote[0].High[i]*adj, dl.InputPrecision))
			datasetAsColumns.AdjLow = append(datasetAsColumns.AdjLow, mathh.Round(yfc.Chart.Result[0].Indicators.Quote[0].Low[i]*adj, dl.InputPrecision))
			datasetAsColumns.AdjClose = append(datasetAsColumns.AdjClose, mathh.Round(yfc.Chart.Result[0].Indicators.AdjClose[0].AdjClose[i], dl.InputPrecision))
			datasetAsColumns.AdjVolume = append(datasetAsColumns.AdjVolume, 0)
		}

		issue := dl.Issue{Symbol: symbol, URL: ucd.URL, DatasetAsColumns: datasetAsColumns}
		group.Issues = append(group.Issues, issue)

		lpf(logh.Info, "Issue loaded; symbol:%5s, StartDate:%s, EndDate:%s, data points:%d",
			issue.Symbol, dateFirst.Format(dl.DateFormat), dateLast.Format(dl.DateFormat), yfcLen-1)
	}

	return group, nil
}
