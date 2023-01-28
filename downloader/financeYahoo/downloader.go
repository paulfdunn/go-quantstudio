package financeYahoo

import (
	"encoding/csv"
	"errors"
	"strings"
	"time"

	dl "github.com/paulfdunn/go-quantstudio/downloader"
	"github.com/paulfdunn/httph"
	"github.com/paulfdunn/logh"
)

var (
	appName string

	yahooURL = "https://query1.finance.yahoo.com/v7/finance/download/%s?period1=%d&period2=%d&" +
		"interval=1d&events=history&includeAdjustedClose=true"
)

func Init(appNameInit string) {
	appName = appNameInit
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
			logh.Map[appName].Printf(logh.Warning, "No urlCollectionDataHeaderIndicesMap entry created for %d, %s", i, v)
		}

		if !wasDefault {
			logh.Map[appName].Printf(logh.Debug, "urlCollectionDataHeaderIndicesMap entry created for %d, %s", i, v)
		}
	}

	if _, ok := urlCollectionDataHeaderIndicesMap["Date"]; !ok {
		logh.Map[appName].Printf(logh.Error, "No Date field in data, records[0]:%+v", urlCollectionDataCSVHeader)
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
			logh.Map[appName].Printf(logh.Error, "the URL is not in map, URL: %s", dl.BaseURL(ucd.URL))
		}

		r := csv.NewReader(strings.NewReader(string(ucd.Bytes)))
		records, err := r.ReadAll()
		if err != nil {
			logh.Map[appName].Printf(logh.Error, "reading Byte(s) from input failed, error:%s", err)
			return nil, err
		}
		if len(records) == 0 {
			logh.Map[appName].Printf(logh.Error, "zero length data, symbol:%5s, body:%s", symbol, string(ucd.Bytes))
			return nil, errors.New("zero length data")
		}

		urlCollectionDataHeaderIndicesMap, err := mapURLCollectionDataHeaderIndices(records[0])
		if err != nil {
			logh.Map[appName].Printf(logh.Error, "could not get column indices from raw data, error:%s", err)
			return nil, err
		}

		orderAsc, dateFirst, dateLast := dl.URLCollectionDataDateOrder(records, urlCollectionDataHeaderIndicesMap)
		data := urlCollectionDataToStructure(symbol, records, urlCollectionDataHeaderIndicesMap, orderAsc)

		issue := dl.Issue{Symbol: symbol, URL: ucd.URL, Dataset: data}
		issue.DatasetAsColumns = issue.ToDatasetAsColumns()
		group.Issues = append(group.Issues, issue)

		logh.Map[appName].Printf(logh.Info, "Issue loaded; symbol:%5s, StartDate:%s, EndDate:%s, data points:%d",
			issue.Symbol, dateFirst.Format(dl.DateFormat), dateLast.Format(dl.DateFormat), len(data))
	}

	return group, nil
}

func urlCollectionDataToStructure(symbol string, records [][]string, urlCollectionDataHeaderIndicesMap map[string]int, orderAsc bool) (data []dl.Data) {
	// Convert the RawData into structured Data.
	data = make([]dl.Data, 0)
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
			logh.Map[appName].Printf(logh.Error, "Cannot parse date:%v", record[urlCollectionDataHeaderIndicesMap["Date"]])
			continue
		}

		floatRecord, err := dl.StringRecordToFloat64Record(record, []int{urlCollectionDataHeaderIndicesMap["Date"]}, symbol)
		if err != nil {
			logh.Map[appName].Printf(logh.Error, "Cannot parse record to float:%+v", record)
			continue
		}
		adj := floatRecord[urlCollectionDataHeaderIndicesMap["AdjClose"]] / floatRecord[urlCollectionDataHeaderIndicesMap["Close"]]

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
	return data
}
