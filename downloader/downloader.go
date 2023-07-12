package downloader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/paulfdunn/goutil"
	"github.com/paulfdunn/httph"
	"github.com/paulfdunn/logh"
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
	// Dataset is row based data, as source data is row based.
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
	InputPrecision = 4

	// err on the side of caution; if yahoo sees to much traffic from the same user, it may block.
	Threads = 1
)

var (
	// earliest date for which data is fetched.
	EarliestDate = time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	LatestDate   = time.Now().AddDate(0, 0, 1).Unix()
	// Used to generate test data
	// EarliestDate = time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	// LatestDate   = time.Date(2022, time.January, 5, 0, 0, 0, 0, time.UTC).Unix()

	URLCollectionTimeout = time.Duration(10 * time.Second)

	appName string
)

func Init(appNameInit string) {
	appName = appNameInit
}

// NewGroup is a factory for Group.
// liveData == true, data is downloaded from Yahoo; otherwise it is loaded from a file saved
// from the prior call.
func NewGroup(liveData bool, dataFilePath string, name string, symbols []string, url string,
	callbackURLCollectionDataToGroup URLCollectionDataToGroup) (*Group, error) {
	updateLatestDate()
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
			logh.Map[appName].Printf(logh.Error, "saving group as csv: %+v", err)
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
	}

	return urls, urlSymbolMap
}

func LoadURLCollectionDataFromFile(dataFilePath string) (urlData []httph.URLCollectionData, err error) {
	logh.Map[appName].Printf(logh.Warning, "Prior data loaded from file.")
	bIn, err := ioutil.ReadFile(dataFilePath + BinaryExtension)
	if err != nil {
		logh.Map[appName].Printf(logh.Error, "reading JSON bodies failed, error:%s", err)
		return nil, err
	}
	err = json.Unmarshal(bIn, &urlData)
	if err != nil {
		logh.Map[appName].Printf(logh.Error, "unmarshaling JSON data failed, error:%s", err)
		return nil, err
	}
	return urlData, nil
}

func StringRecordToFloat64Record(record []string, skipIndices []int, symbol string) (out []float64, nulls int, err error) {
	out = make([]float64, len(record))
	for i, v := range record {
		if goutil.InIntSlice(i, skipIndices) {
			continue
		}
		if v == "null" {
			logh.Map[appName].Printf(logh.Debug, "null value in record, cannot parse to float, symbol: %s, record:%+v", symbol, record)
			nulls++
			continue
		}
		out[i], err = strconv.ParseFloat(v, 64)
		out[i] = goutil.Round(out[i], InputPrecision)
		if err != nil {
			err := fmt.Errorf("converting to float, err%v\nsymbol: %s, record:%+v", err, symbol, record)
			logh.Map[appName].Printf(logh.Error, "%+v", err)
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
		logh.Map[appName].Printf(logh.Error, "opening csn file for output:%+v", err)
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, issue := range grp.Issues {
		header := "symbol,   Date,  Open,  High,  Low,  Close,  Volume,  AdjOpen,  AdjHigh,  AdjLow,  AdjClose,  AdjVolume"
		_, err := w.WriteString(header + "\n")
		if err != nil {
			logh.Map[appName].Printf(logh.Error, "writing data header:%+v", err)
			return err
		}
		for _, data := range issue.Dataset {
			_, err := w.WriteString(fmt.Sprintf("%s, %s, %10.4f, %10.4f, %10.4f, %10.4f, %15.4f, %10.4f, %10.4f, %10.4f, %10.4f, %15.4f\n",
				issue.Symbol, data.Date.Format(DateFormat),
				data.Open, data.High, data.Low, data.Close, data.Volume,
				data.AdjOpen, data.AdjHigh, data.AdjLow, data.AdjClose, data.AdjVolume))
			if err != nil {
				logh.Map[appName].Printf(logh.Error, "writing data:%+v", err)
				return err
			}
		}
	}
	w.Flush()
	return nil
}

func (dt Data) String() string {
	out, err := json.MarshalIndent(dt, "", "  ")
	logh.Map[appName].Printf(logh.Error, "calling json.MarshalIndent: %s", err)
	return string(goutil.PrettyJSON(out))
}

func (dac DatasetAsColumns) String() string {
	out, err := json.MarshalIndent(dac, "", "  ")
	logh.Map[appName].Printf(logh.Error, "calling json.MarshalIndent: %s", err)
	return string(goutil.PrettyJSON(out))
}

func (grp Group) String() string {
	out, err := json.MarshalIndent(grp, "", "  ")
	logh.Map[appName].Printf(logh.Error, "calling json.MarshalIndent: %s", err)
	return string(goutil.PrettyJSON(out))
}

func (iss Issue) String() string {
	out, err := json.MarshalIndent(iss, "", "  ")
	logh.Map[appName].Printf(logh.Error, "calling json.MarshalIndent: %s", err)
	return string(goutil.PrettyJSON(out))
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
	urlData := httph.CollectURLs(urls, URLCollectionTimeout, http.MethodGet, Threads)
	// Response cannot be Marshalled; set to nil to prevent errors when saving the data
	// and also generating SA1026 lint error.
	for i := range urlData {
		urlData[i].Response = nil
	}
	return urlData
}

func saveURLCollectionData(urlData []httph.URLCollectionData, dataFilePath string) error {
	//lint:ignore SA1026 request was set to nil in collectGroup to avoid problem
	bOut, err := json.Marshal(urlData)
	if err != nil {
		logh.Map[appName].Printf(logh.Error, "marshalling JSON bodies failed, error:%s", err)
		return err
	}
	err = ioutil.WriteFile(dataFilePath+BinaryExtension, bOut, 0644)
	if err != nil {
		logh.Map[appName].Printf(logh.Error, "writing JSON bodies failed, error:%s", err)
		return err
	}
	return nil
}

func updateLatestDate() {
	LatestDate = time.Now().AddDate(0, 0, 1).Unix()
}
