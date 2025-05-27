package financeYahooChart

import (
	"fmt"
	"testing"
	"time"

	dl "github.com/paulfdunn/go-quantstudio/downloader"
)

const (
	testDataFilePath = "./test/test_etf"
)

var (
	testSymbols = []string{"dia", "qqq"}
)

func init() {
	Init("test")
	dl.Init("test")
	// For testing, override latestDate so it is a fixed value. Otherwise
	// it changes every day and the tests fail.
	dl.EarliestDate = time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	dl.LatestDate = time.Date(2022, time.January, 5, 0, 0, 0, 0, time.UTC).Unix()
}

func ExampleNewGroup() {
	// For Example purposes liveData is set false. Programmatic callers can set this true
	// to download data and get a Group returned for that data, or set the value to false
	// to reprocess previously downloaded data.

	liveData := true
	group, err := NewGroup(liveData, testDataFilePath, "testGroup", testSymbols)
	if err != nil {
		var t *testing.T
		t.Error()
	}

	fmt.Printf("%+v", group.String())
	// Output:
	// {
	//   "Name": "testGroup",
	//   "Issues": [
	//     {
	//       "Symbol": "dia",
	//       "URL": "https://query2.finance.yahoo.com/v8/finance/chart/dia?period1=1640995200\u0026period2=1641340800\u0026interval=1d",
	//       "Dataset": null,
	//       "DatasetAsColumns": {
	//         "Date": [
	//           "2022-01-03T07:30:00-07:00",
	//           "2022-01-04T07:30:00-07:00"
	//         ],
	//         "Open": [364.3399963378906,367.3399963378906],
	//         "High": [365.8500061035156,369.2099914550781],
	//         "Low": [362.29998779296875,367.2099914550781],
	//         "Close": [365.67999267578125,367.8699951171875],
	//         "Volume": [5624100,5462200],
	//         "AdjOpen": [342.73,345.55],
	//         "AdjHigh": [344.15,347.31],
	//         "AdjLow": [340.81,345.43],
	//         "AdjClose": [343.99,346.05],
	//         "AdjVolume": [0,0]
	//       }
	//     },
	//     {
	//       "Symbol": "qqq",
	//       "URL": "https://query2.finance.yahoo.com/v8/finance/chart/qqq?period1=1640995200\u0026period2=1641340800\u0026interval=1d",
	//       "Dataset": null,
	//       "DatasetAsColumns": {
	//         "Date": [
	//           "2022-01-03T07:30:00-07:00",
	//           "2022-01-04T07:30:00-07:00"
	//         ],
	//         "Open": [399.04998779296875,402.239990234375],
	//         "High": [401.94000244140625,402.2799987792969],
	//         "Low": [396.8800048828125,393.2900085449219],
	//         "Close": [401.67999267578125,396.4700012207031],
	//         "Volume": [40575900,58027200],
	//         "AdjOpen": [390.51,393.63],
	//         "AdjHigh": [393.33,393.67],
	//         "AdjLow": [388.38,384.87],
	//         "AdjClose": [393.08,387.98],
	//         "AdjVolume": [0,0]
	//       }
	//     }
	//   ]
	// }
}
