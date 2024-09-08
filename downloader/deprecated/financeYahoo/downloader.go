// Package financeYahoo implements a price downloader using the finance.yahoo API.
// As of SEP-6-2024 that API requires a fee to use, and this code is deprecated.
// See package financeYahooChart for the replacement.
package financeYahoo

import (
	"encoding/csv"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/paulfdunn/go-helper/logh"
	"github.com/paulfdunn/go-helper/neth/httph"
	dl "github.com/paulfdunn/go-quantstudio/downloader"
)

var (
	appName string
	// lp      func(level logh.LoghLevel, v ...interface{})
	lpf func(level logh.LoghLevel, format string, v ...interface{})

	yahooURL = "https://query1.finance.yahoo.com/v7/finance/download/%s?period1=%d&period2=%d&" +
		"interval=1d&events=history&includeAdjustedClose=true"
)

func Init(appNameInit string) {
	appName = appNameInit
	// lp = logh.Map[appName].Println
	lpf = logh.Map[appName].Printf
}

func NewGroup(liveData bool, dataFilePath string, name string, symbols []string) (*dl.Group, error) {
	return dl.NewGroup(liveData, dataFilePath, name, symbols, yahooURL, urlCollectionDataToGroup)
}

// mapURLCollectionDataHeaderIndices makes a map of column names to struct members.
func mapURLCollectionDataHeaderIndices(urlCollectionDataCSVHeader []string) (urlCollectionDataHeaderIndicesMap map[string]int, err error) {
	urlCollectionDataHeaderIndicesMap = make(map[string]int)
	for i, v := range urlCollectionDataCSVHeader {
		wasDefault := false
		switch v {
		case "Date":
			urlCollectionDataHeaderIndicesMap["Date"] = i
		case "Open":
			urlCollectionDataHeaderIndicesMap["Open"] = i
		case "High":
			urlCollectionDataHeaderIndicesMap["High"] = i
		case "Low":
			urlCollectionDataHeaderIndicesMap["Low"] = i
		case "Close":
			urlCollectionDataHeaderIndicesMap["Close"] = i
		case "Volume":
			urlCollectionDataHeaderIndicesMap["Volume"] = i
		case "Adj Close":
			urlCollectionDataHeaderIndicesMap["AdjClose"] = i
		default:
			wasDefault = true
			lpf(logh.Warning, "No urlCollectionDataHeaderIndicesMap entry created for %d, %s", i, v)
		}

		if !wasDefault {
			lpf(logh.Debug, "urlCollectionDataHeaderIndicesMap entry created for %d, %s", i, v)
		}
	}

	if _, ok := urlCollectionDataHeaderIndicesMap["Date"]; !ok {
		lpf(logh.Error, "no Date field in data, records[0]:%+v", urlCollectionDataCSVHeader)
		return nil, err
	}

	return urlCollectionDataHeaderIndicesMap, nil
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

		lpf(logh.Info, "%s", string(ucd.Bytes))
		r := csv.NewReader(strings.NewReader(string(ucd.Bytes)))
		records, err := r.ReadAll()
		if err != nil {
			lpf(logh.Error, "reading Byte(s) from input failed, symbol: %s, error:%s", symbol, err)
			return nil, err
		}
		if len(records) == 0 {
			lpf(logh.Error, "zero length data, symbol:%5s, body:%s", symbol, string(ucd.Bytes))
			return nil, errors.New("zero length data")
		}

		urlCollectionDataHeaderIndicesMap, err := mapURLCollectionDataHeaderIndices(records[0])
		if err != nil {
			lpf(logh.Error, "could not get column indices from raw data, error:%s", err)
			return nil, err
		}

		orderAsc, dateFirst, dateLast := dl.URLCollectionDataDateOrder(records, urlCollectionDataHeaderIndicesMap)
		data := urlCollectionDataToStructure(symbol, records, urlCollectionDataHeaderIndicesMap, orderAsc)

		issue := dl.Issue{Symbol: symbol, URL: ucd.URL, Dataset: data}
		issue.DatasetAsColumns = issue.ToDatasetAsColumns()
		group.Issues = append(group.Issues, issue)

		lpf(logh.Info, "Issue loaded; symbol:%5s, StartDate:%s, EndDate:%s, data points:%d",
			issue.Symbol, dateFirst.Format(dl.DateFormat), dateLast.Format(dl.DateFormat), len(data))
	}

	return group, nil
}

func urlCollectionDataToStructure(symbol string, records [][]string, urlCollectionDataHeaderIndicesMap map[string]int, orderAsc bool) (data []dl.Data) {
	// Convert the RawData into structured Data.
	data = make([]dl.Data, 0)
	cummulativeNulls := 0
	for i := range records {
		// Skip the header row.
		if i == 0 {
			continue
		}
		// Always process from earliest date to latest.
		var index int
		if orderAsc {
			index = i
		} else {
			index = len(records) - 1 - i
		}
		record := records[index]

		date, err := time.Parse(dl.DateFormat, record[urlCollectionDataHeaderIndicesMap["Date"]])
		if err != nil {
			lpf(logh.Error, "cannot parse date:%v", record[urlCollectionDataHeaderIndicesMap["Date"]])
			continue
		}

		floatRecord, nulls, err := dl.StringRecordToFloat64Record(record, []int{urlCollectionDataHeaderIndicesMap["Date"]}, symbol)
		cummulativeNulls += nulls
		if err != nil {
			lpf(logh.Error, "cannot parse record to float, symbol: %s, record:%+v", symbol, record)
			continue
		}
		adj := floatRecord[urlCollectionDataHeaderIndicesMap["AdjClose"]] / floatRecord[urlCollectionDataHeaderIndicesMap["Close"]]
		// Some issues have weekend and holiday dates with price data that is all zeros.
		if math.IsNaN(adj) {
			if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
				continue
			}
			lpf(logh.Warning, "adj is NaN (Close was 0) on weekday (may be holiday), symbol: %s, date: %s", symbol, date)
			continue
		}

		data = append(data, dl.Data{
			Date:     date,
			Open:     floatRecord[urlCollectionDataHeaderIndicesMap["Open"]],
			High:     floatRecord[urlCollectionDataHeaderIndicesMap["High"]],
			Low:      floatRecord[urlCollectionDataHeaderIndicesMap["Low"]],
			Close:    floatRecord[urlCollectionDataHeaderIndicesMap["Close"]],
			Volume:   floatRecord[urlCollectionDataHeaderIndicesMap["Volume"]],
			AdjOpen:  floatRecord[urlCollectionDataHeaderIndicesMap["Open"]] * adj,
			AdjHigh:  floatRecord[urlCollectionDataHeaderIndicesMap["High"]] * adj,
			AdjLow:   floatRecord[urlCollectionDataHeaderIndicesMap["Low"]] * adj,
			AdjClose: floatRecord[urlCollectionDataHeaderIndicesMap["AdjClose"]],
		})
	}

	if cummulativeNulls > 0 {
		lpf(logh.Warning, "symbol: %s, total null values in records: %d", symbol, cummulativeNulls)
	}

	return data
}
