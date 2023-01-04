package financeYahoo

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
	// For testing, override latestDate so it is a fixed value. Otherwise
	// it changes every day and the tests fail.
	dl.LatestDate = time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
}

func ExampleNewGroup() {
	// For Example purposes liveData is set false. Programmatic callers can set this true
	// to download data and get a Group returned for that data, or set the value to false
	// to reprocess previously downloaded data.
	liveData := false
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
	//       "URL": "https://query1.finance.yahoo.com/v7/finance/download/dia?period1=1640995200\u0026period2=1641340800\u0026interval=1d\u0026events=history\u0026includeAdjustedClose=true",
	//       "Dataset": [
	//         {
	//           "Date": "2022-01-03T00:00:00Z",
	//           "Open": 364.34,
	//           "High": 365.85,
	//           "Low": 362.3,
	//           "Close": 365.68,
	//           "Volume": 5624100,
	//           "AdjOpen": 357.3278897232553,
	//           "AdjHigh": 358.8088281694378,
	//           "AdjLow": 355.3271516900022,
	//           "AdjClose": 358.6421,
	//           "AdjVolume": 0
	//         },
	//         {
	//           "Date": "2022-01-04T00:00:00Z",
	//           "Open": 367.34,
	//           "High": 369.21,
	//           "Low": 367.21,
	//           "Close": 367.87,
	//           "Volume": 5462200,
	//           "AdjOpen": 360.2702003425123,
	//           "AdjHigh": 362.10421045478023,
	//           "AdjLow": 360.1427023133172,
	//           "AdjClose": 360.79,
	//           "AdjVolume": 0
	//         }
	//       ],
	//       "DatasetAsColumns": {
	//         "Date": [
	//           "2022-01-03T00:00:00Z",
	//           "2022-01-04T00:00:00Z"
	//         ],
	//         "Open": [364.34,367.34],
	//         "High": [365.85,369.21],
	//         "Low": [362.3,367.21],
	//         "Close": [365.68,367.87],
	//         "Volume": [5624100,5462200],
	//         "AdjOpen": [357.3278897232553,360.2702003425123],
	//         "AdjHigh": [358.8088281694378,362.10421045478023],
	//         "AdjLow": [355.3271516900022,360.1427023133172],
	//         "AdjClose": [358.6421,360.79],
	//         "AdjVolume": [0,0]
	//       }
	//     },
	//     {
	//       "Symbol": "qqq",
	//       "URL": "https://query1.finance.yahoo.com/v7/finance/download/qqq?period1=1640995200\u0026period2=1641340800\u0026interval=1d\u0026events=history\u0026includeAdjustedClose=true",
	//       "Dataset": [
	//         {
	//           "Date": "2022-01-03T00:00:00Z",
	//           "Open": 399.05,
	//           "High": 401.94,
	//           "Low": 396.88,
	//           "Close": 401.68,
	//           "Volume": 40575900,
	//           "AdjOpen": 396.1306404849632,
	//           "AdjHigh": 398.9994978988249,
	//           "AdjLow": 393.9765157140012,
	//           "AdjClose": 398.7414,
	//           "AdjVolume": 0
	//         },
	//         {
	//           "Date": "2022-01-04T00:00:00Z",
	//           "Open": 402.24,
	//           "High": 402.28,
	//           "Low": 393.29,
	//           "Close": 396.47,
	//           "Volume": 58027200,
	//           "AdjOpen": 399.2972877645219,
	//           "AdjHigh": 399.3369951320402,
	//           "AdjLow": 390.4127642822912,
	//           "AdjClose": 393.5695,
	//           "AdjVolume": 0
	//         }
	//       ],
	//       "DatasetAsColumns": {
	//         "Date": [
	//           "2022-01-03T00:00:00Z",
	//           "2022-01-04T00:00:00Z"
	//         ],
	//         "Open": [399.05,402.24],
	//         "High": [401.94,402.28],
	//         "Low": [396.88,393.29],
	//         "Close": [401.68,396.47],
	//         "Volume": [40575900,58027200],
	//         "AdjOpen": [396.1306404849632,399.2972877645219],
	//         "AdjHigh": [398.9994978988249,399.3369951320402],
	//         "AdjLow": [393.9765157140012,390.4127642822912],
	//         "AdjClose": [398.7414,393.5695],
	//         "AdjVolume": [0,0]
	//       }
	//     }
	//   ]
	// }
}

// This test includes calls to mapURLCollectionDataHeaderIndices, urlCollectionDataToStructure,
// and urlCollectionDataDateOrder via a call to urlCollectionDataToGroup
func Example_loadURLCollectionDataFromFile() {
	_, urlSymbolMap := dl.GenerateURLs(testSymbols, yahooURL)
	urlData, _ := dl.LoadURLCollectionDataFromFile(testDataFilePath)
	group, _ := urlCollectionDataToGroup(urlData, urlSymbolMap, "test")

	fmt.Printf("%+v", group.String())

	// Output:
	// {
	//   "Name": "test",
	//   "Issues": [
	//     {
	//       "Symbol": "dia",
	//       "URL": "https://query1.finance.yahoo.com/v7/finance/download/dia?period1=1640995200\u0026period2=1641340800\u0026interval=1d\u0026events=history\u0026includeAdjustedClose=true",
	//       "Dataset": [
	//         {
	//           "Date": "2022-01-03T00:00:00Z",
	//           "Open": 364.34,
	//           "High": 365.85,
	//           "Low": 362.3,
	//           "Close": 365.68,
	//           "Volume": 5624100,
	//           "AdjOpen": 357.3278897232553,
	//           "AdjHigh": 358.8088281694378,
	//           "AdjLow": 355.3271516900022,
	//           "AdjClose": 358.6421,
	//           "AdjVolume": 0
	//         },
	//         {
	//           "Date": "2022-01-04T00:00:00Z",
	//           "Open": 367.34,
	//           "High": 369.21,
	//           "Low": 367.21,
	//           "Close": 367.87,
	//           "Volume": 5462200,
	//           "AdjOpen": 360.2702003425123,
	//           "AdjHigh": 362.10421045478023,
	//           "AdjLow": 360.1427023133172,
	//           "AdjClose": 360.79,
	//           "AdjVolume": 0
	//         }
	//       ],
	//       "DatasetAsColumns": {
	//         "Date": [
	//           "2022-01-03T00:00:00Z",
	//           "2022-01-04T00:00:00Z"
	//         ],
	//         "Open": [364.34,367.34],
	//         "High": [365.85,369.21],
	//         "Low": [362.3,367.21],
	//         "Close": [365.68,367.87],
	//         "Volume": [5624100,5462200],
	//         "AdjOpen": [357.3278897232553,360.2702003425123],
	//         "AdjHigh": [358.8088281694378,362.10421045478023],
	//         "AdjLow": [355.3271516900022,360.1427023133172],
	//         "AdjClose": [358.6421,360.79],
	//         "AdjVolume": [0,0]
	//       }
	//     },
	//     {
	//       "Symbol": "qqq",
	//       "URL": "https://query1.finance.yahoo.com/v7/finance/download/qqq?period1=1640995200\u0026period2=1641340800\u0026interval=1d\u0026events=history\u0026includeAdjustedClose=true",
	//       "Dataset": [
	//         {
	//           "Date": "2022-01-03T00:00:00Z",
	//           "Open": 399.05,
	//           "High": 401.94,
	//           "Low": 396.88,
	//           "Close": 401.68,
	//           "Volume": 40575900,
	//           "AdjOpen": 396.1306404849632,
	//           "AdjHigh": 398.9994978988249,
	//           "AdjLow": 393.9765157140012,
	//           "AdjClose": 398.7414,
	//           "AdjVolume": 0
	//         },
	//         {
	//           "Date": "2022-01-04T00:00:00Z",
	//           "Open": 402.24,
	//           "High": 402.28,
	//           "Low": 393.29,
	//           "Close": 396.47,
	//           "Volume": 58027200,
	//           "AdjOpen": 399.2972877645219,
	//           "AdjHigh": 399.3369951320402,
	//           "AdjLow": 390.4127642822912,
	//           "AdjClose": 393.5695,
	//           "AdjVolume": 0
	//         }
	//       ],
	//       "DatasetAsColumns": {
	//         "Date": [
	//           "2022-01-03T00:00:00Z",
	//           "2022-01-04T00:00:00Z"
	//         ],
	//         "Open": [399.05,402.24],
	//         "High": [401.94,402.28],
	//         "Low": [396.88,393.29],
	//         "Close": [401.68,396.47],
	//         "Volume": [40575900,58027200],
	//         "AdjOpen": [396.1306404849632,399.2972877645219],
	//         "AdjHigh": [398.9994978988249,399.3369951320402],
	//         "AdjLow": [393.9765157140012,390.4127642822912],
	//         "AdjClose": [398.7414,393.5695],
	//         "AdjVolume": [0,0]
	//       }
	//     }
	//   ]
	// }
}
