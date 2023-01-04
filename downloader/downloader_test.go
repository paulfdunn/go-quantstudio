package downloader

import (
	"fmt"
	"time"
)

var (
	testSymbols = []string{"dia", "qqq"}
	testURL     = "https://query1.finance.yahoo.com/v7/finance/download/%s?period1=%d&period2=%d&" +
		"interval=1d&events=history&includeAdjustedClose=true"
)

func init() {
	// For testing, override latestDate so it is a fixed value. Otherwise
	// it changes every day and the tests fail.
	LatestDate = time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
}

func ExampleBaseURL() {
	urls, _ := GenerateURLs([]string{"qqq"}, testURL)
	fmt.Printf("%s\n", urls[0])
	fmt.Printf("%s\n", BaseURL(urls[0]))

	// Output:
	// https://query1.finance.yahoo.com/v7/finance/download/qqq?period1=631152000&period2=1672531200&interval=1d&events=history&includeAdjustedClose=true
	// https://query1.finance.yahoo.com/v7/finance/download/qqq
}

func Example_generateURLs() {
	urls, urlSymbolMap := GenerateURLs(testSymbols, testURL)
	for _, url := range urls {
		fmt.Printf("%s\n", url)
	}
	fmt.Printf("urlSymbolMap: %+v", urlSymbolMap)

	// Output:
	// https://query1.finance.yahoo.com/v7/finance/download/dia?period1=631152000&period2=1672531200&interval=1d&events=history&includeAdjustedClose=true
	// https://query1.finance.yahoo.com/v7/finance/download/qqq?period1=631152000&period2=1672531200&interval=1d&events=history&includeAdjustedClose=true
	// urlSymbolMap: map[https://query1.finance.yahoo.com/v7/finance/download/dia:dia https://query1.finance.yahoo.com/v7/finance/download/qqq:qqq]
}
