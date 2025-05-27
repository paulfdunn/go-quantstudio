package downloader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/paulfdunn/go-helper/encodingh/v2/jsonh"
	"github.com/paulfdunn/go-helper/logh/v2"
	"github.com/paulfdunn/go-helper/mathh/v2"
	"github.com/paulfdunn/go-helper/neth/v2/httph"
)

type Downloader interface {
	NewGroup(liveData bool, dataFilePath string, name string, symbols []string) (*Group, error)
}

// Group is a collection of Issues (stocks, ETFs, etc.)
type Group struct {
	Name   string
	Issues []Issue
}

// Issue is an item for which Data is collected. I.E. a stock, ETF, etc.
type Issue struct {
	Symbol string
	URL    string
	// Dataset is row based data for data sources in that format; convert to DatasetAsColumns
	// using ToDatasetAsColumns().
	Dataset []Data
	// DatasetAsColumns is column based data as that is sometimes easier to work with.
	DatasetAsColumns DatasetAsColumns
}

// Data is used to Unmarshal data. This structure must
// match the JSON returned from finance.yahoo.
// Returned data is always sorted in Date ascending order.
type Data struct {
	Date      time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	AdjOpen   float64
	AdjHigh   float64
	AdjLow    float64
	AdjClose  float64
	AdjVolume float64
}

type DatasetAsColumns struct {
	Date      []time.Time
	Open      []float64
	High      []float64
	Low       []float64
	Close     []float64
	Volume    []float64
	AdjOpen   []float64
	AdjHigh   []float64
	AdjLow    []float64
	AdjClose  []float64
	AdjVolume []float64
}

type URLCollectionDataToGroup func(urlData []httph.URLCollectionData, urlSymbolMap map[string]string, name string) (group *Group, err error)

const (
	BinaryExtension = ".bin"
	CSVExtension    = ".csv"
	DateFormat      = "2006-01-02"

	// Floats are rounded to this number of decimal points. Yahoo will slightly alter some values
	// with each call, but only at very high numbers of decimal places. That makes debugging
	// difficult.
	InputPrecision = 2

	// err on the side of caution; if yahoo sees to much traffic from the same user, it may block.
	Threads = 1
)

var (
	appName string
	// lp      func(level logh.LoghLevel, v ...interface{})
	lpf func(level logh.LoghLevel, format string, v ...interface{})

	// earliest date for which data is fetched.
	EarliestDate = time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	LatestDate   = time.Now().AddDate(0, 0, 1).Unix()

	URLCollectionTimeout = time.Duration(10 * time.Second)
)

func Init(appNameInit string) {
	appName = appNameInit
	// lp = logh.Map[appName].Println
	lpf = logh.Map[appName].Printf
}

// NewGroup is a factory for Group.
// liveData == true, data is downloaded from Yahoo; otherwise it is loaded from a file saved
// from the prior call.
func NewGroup(liveData bool, dataFilePath string, name string, symbols []string, url string,
	callbackURLCollectionDataToGroup URLCollectionDataToGroup) (*Group, error) {
	// updateLatestDate()
	urls, urlSymbolMap := GenerateURLs(symbols, url)

	var urlData []httph.URLCollectionData
	if liveData {
		// Get data for all symbols and save it.
		urlData = collectGroup(urls)
		err := saveURLCollectionData(urlData, dataFilePath)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		if urlData, err = LoadURLCollectionDataFromFile(dataFilePath); err != nil {
			return nil, err
		}
	}

	group, err := callbackURLCollectionDataToGroup(urlData, urlSymbolMap, name)
	if err != nil {
		return nil, err
	}
	if liveData {
		err := group.SaveCSV(dataFilePath)
		if err != nil {
			lpf(logh.Error, "saving group as csv: %+v", err)
			return nil, err
		}
	}
	return group, nil
}

func BaseURL(url string) string {
	return strings.Split(url, "?")[0]
}

// GenerateURLs generates URLs for the symbols as well as a symbol to URL map for
// use with data returned from collectGroup
func GenerateURLs(symbols []string, url string) (urls []string, urlSymbolMap map[string]string) {
	// Make the URLs from which to fetch data.
	urls = make([]string, 0, len(symbols))
	// Keep a map of base URL to symbol.
	urlSymbolMap = make(map[string]string)
	for _, s := range symbols {
		url := fmt.Sprintf(url, s, EarliestDate, LatestDate)
		urls = append(urls, url)
		urlSymbolMap[BaseURL(url)] = s
		lpf(logh.Debug, "Symbol: %s, URL: %s", s, url)
	}

	return urls, urlSymbolMap
}

func LoadURLCollectionDataFromFile(dataFilePath string) (urlData []httph.URLCollectionData, err error) {
	lpf(logh.Warning, "Prior data loaded from file.")
	bIn, err := os.ReadFile(dataFilePath + BinaryExtension)
	if err != nil {
		lpf(logh.Error, "reading JSON bodies failed, error:%s", err)
		return nil, err
	}
	err = json.Unmarshal(bIn, &urlData)
	if err != nil {
		lpf(logh.Error, "unmarshaling JSON data failed, error:%s", err)
		return nil, err
	}
	return urlData, nil
}

func StringRecordToFloat64Record(record []string, skipIndices []int, symbol string) (out []float64, nulls int, err error) {
	out = make([]float64, len(record))
	for i, v := range record {
		if slices.Contains(skipIndices, i) {
			continue
		}
		if v == "null" {
			lpf(logh.Debug, "null value in record, cannot parse to float, symbol: %s, record:%+v", symbol, record)
			nulls++
			continue
		}
		out[i], err = strconv.ParseFloat(v, 64)
		out[i] = mathh.Round(out[i], InputPrecision)
		if err != nil {
			err := fmt.Errorf("converting to float, err%v\nsymbol: %s, record:%+v", err, symbol, record)
			lpf(logh.Error, "%+v", err)
			return nil, nulls, err
		}
	}
	return out, nulls, nil
}

func URLCollectionDataDateOrder(records [][]string, urlCollectionDataHeaderIndicesMap map[string]int) (orderAsc bool,
	dateFirst time.Time, dateLast time.Time) {
	rawLength := len(records)
	rFirst := records[1]
	rLast := records[rawLength-1]
	dateFirst, _ = time.Parse(DateFormat, rFirst[urlCollectionDataHeaderIndicesMap["Date"]])
	dateLast, _ = time.Parse(DateFormat, rLast[urlCollectionDataHeaderIndicesMap["Date"]])
	orderAsc = true
	if dateFirst.After(dateLast) {
		orderAsc = false
	}
	return orderAsc, dateFirst, dateLast
}

func (grp Group) SaveCSV(dataFilePath string) error {
	f, err := os.Create(dataFilePath + CSVExtension)
	if err != nil {
		lpf(logh.Error, "opening CSV file for output:%+v", err)
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, issue := range grp.Issues {
		header := "symbol,   Date,  Open,  High,  Low,  Close,  Volume,  AdjOpen,  AdjHigh,  AdjLow,  AdjClose,  AdjVolume"
		_, err := w.WriteString(header + "\n")
		if err != nil {
			lpf(logh.Error, "writing data header:%+v", err)
			return err
		}
		for indx := range issue.DatasetAsColumns.Date {
			_, err := w.WriteString(fmt.Sprintf("%s, %s, %10.4f, %10.4f, %10.4f, %10.4f, %15.4f, %10.4f, %10.4f, %10.4f, %10.4f, %15.4f\n",
				issue.Symbol, issue.DatasetAsColumns.Date[indx].Format(DateFormat),
				issue.DatasetAsColumns.Open[indx], issue.DatasetAsColumns.High[indx], issue.DatasetAsColumns.Low[indx], issue.DatasetAsColumns.Close[indx], issue.DatasetAsColumns.Volume[indx],
				issue.DatasetAsColumns.AdjOpen[indx], issue.DatasetAsColumns.AdjHigh[indx], issue.DatasetAsColumns.AdjLow[indx], issue.DatasetAsColumns.AdjClose[indx], issue.DatasetAsColumns.AdjVolume[indx]))
			if err != nil {
				lpf(logh.Error, "writing data:%+v", err)
				return err
			}
		}
	}
	w.Flush()
	return nil
}

func (dt Data) String() string {
	out, err := json.MarshalIndent(dt, "", "  ")
	lpf(logh.Error, "calling json.MarshalIndent: %s", err)
	return string(jsonh.PrettyJSON(out))
}

func (dac DatasetAsColumns) String() string {
	out, err := json.MarshalIndent(dac, "", "  ")
	lpf(logh.Error, "calling json.MarshalIndent: %s", err)
	return string(jsonh.PrettyJSON(out))
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

// ToDatasetAsColumns converts an issue from a row format to column format
func (iss Issue) ToDatasetAsColumns() DatasetAsColumns {
	Date := make([]time.Time, len(iss.Dataset))
	Open := make([]float64, len(iss.Dataset))
	High := make([]float64, len(iss.Dataset))
	Low := make([]float64, len(iss.Dataset))
	Close := make([]float64, len(iss.Dataset))
	Volume := make([]float64, len(iss.Dataset))
	AdjOpen := make([]float64, len(iss.Dataset))
	AdjHigh := make([]float64, len(iss.Dataset))
	AdjLow := make([]float64, len(iss.Dataset))
	AdjClose := make([]float64, len(iss.Dataset))
	AdjVolume := make([]float64, len(iss.Dataset))
	out := DatasetAsColumns{
		Date: Date, Open: Open, High: High, Low: Low, Close: Close, Volume: Volume,
		AdjOpen: AdjOpen, AdjHigh: AdjHigh, AdjLow: AdjLow, AdjClose: AdjClose, AdjVolume: AdjVolume}

	for i, v := range iss.Dataset {
		Date[i] = v.Date
		Open[i] = v.Open
		High[i] = v.High
		Low[i] = v.Low
		Close[i] = v.Close
		Volume[i] = v.Volume
		AdjOpen[i] = v.AdjOpen
		AdjHigh[i] = v.AdjHigh
		AdjLow[i] = v.AdjLow
		AdjClose[i] = v.AdjClose
		AdjVolume[i] = v.AdjVolume
	}

	return out
}

func collectGroup(urls []string) []httph.URLCollectionData {
	// Get data for all symbols.
	headers := []httph.Header{
		{Key: "User-Agent", Value: "Golang_Spider_Bot/3.0"},
	}
	urlData := httph.CollectURLs(urls, URLCollectionTimeout, http.MethodGet, Threads, headers)
	// Response cannot be Marshalled; set to nil to prevent errors when saving the data
	// and also generating SA1026 lint error.
	for i := range urlData {
		urlData[i].Response = nil
	}
	return urlData
}

func saveURLCollectionData(urlData []httph.URLCollectionData, dataFilePath string) error {
	//lint:ignore SA1026 request was set to nil in collectGroup to avoid problem
	//nolint:staticcheck
	bOut, err := json.Marshal(urlData)
	if err != nil {
		lpf(logh.Error, "marshalling JSON bodies failed, error:%s", err)
		return err
	}
	err = os.WriteFile(dataFilePath+BinaryExtension, bOut, 0644)
	if err != nil {
		lpf(logh.Error, "writing JSON bodies failed, error:%s", err)
		return err
	}
	return nil
}

// func updateLatestDate() {
// 	LatestDate = time.Now().AddDate(0, 0, 1).Unix()
// }
