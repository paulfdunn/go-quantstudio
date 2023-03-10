package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/paulfdunn/go-quantstudio/defs"
	"github.com/paulfdunn/go-quantstudio/downloader"
	"github.com/paulfdunn/go-quantstudio/downloader/financeYahoo"
	"github.com/paulfdunn/go-quantstudio/quant"
	"github.com/paulfdunn/logh"
)

var (
	// Only needed to shorten the log statements
	appName = defs.AppName

	// CLI flags
	liveDataPtr, runRangePtr                *bool
	groupNamePtr, logFilePtr, symbolCSVList *string
	logLevel                                *int

	// dataDirectorySuffix is appended to the users home directory.
	dataDirectorySuffix = filepath.Join(`tmp`, appName)
	dataDirectory       string

	dlGroupChan chan *downloader.Group

	//go:embed assets/chart.js assets/index.html assets/plotly-2.16.1.min.js
	staticFS embed.FS
)

func crashDetect() {
	if err := recover(); err != nil {
		errOut := fmt.Sprintf("panic: %+v\n%+v", err, string(debug.Stack()))
		fmt.Println(errOut)
		logh.Map[appName].Println(logh.Error, errOut)
		errShutdown := logh.ShutdownAll()
		if errShutdown != nil {
			logh.Map[appName].Printf(logh.Error, fmt.Sprintf("%#v", errShutdown))
		}
	}
}

func Init() {
	defer crashDetect()

	usr, err := user.Current()
	if err != nil {
		fmt.Printf("Error getting user.Currrent: %+v", err)
	}

	dataDirectory = filepath.Join(usr.HomeDir, dataDirectorySuffix)
	err = os.MkdirAll(dataDirectory, 0777)
	if err != nil {
		fmt.Printf("Error creating data directory: : %+v", err)
	}

	// CLI flags
	groupNamePtr = flag.String("groupname", "ETFs", "Name for this group of symbols. Used for naming output files when processing groups of symbols. I.E. maybe you want to download/analyze stocks separately from ETFs")
	liveDataPtr = flag.Bool("livedata", true, "Get live data; otherwise load from file created during prior call. (Using the download button in the GUI will ALWAYS download new data.)")
	logFilePtr = flag.String("logfile", "", "Name of log file in "+dataDirectory+"; blank to print logs to terminal.")
	logLevel = flag.Int("loglevel", int(logh.Info), fmt.Sprintf("Logging level; default %d. Zero based index into: %v",
		int(logh.Info), logh.DefaultLevels))
	runRangePtr = flag.Bool("runrange", false, "When true, runs a range of parameters and exits.")
	symbolCSVList = flag.String("symbolCSVList", defs.SymbolsDefault, "Comma separated list of symbols for which to download prices")
	flag.Parse()

	var logFilepath string
	if *logFilePtr != "" {
		logFilepath = filepath.Join(dataDirectory, *logFilePtr)
	}
	logh.New(appName, logFilepath, logh.DefaultLevels, logh.LoghLevel(*logLevel),
		logh.DefaultFlags, 100, int64(10e6))

	logh.Map[appName].Printf(logh.Debug, "user.Current(): %+v", usr)
	logh.Map[appName].Printf(logh.Info, "Data and logs being saved to directory: %s", dataDirectory)

	downloader.Init(appName)
	quant.Init(appName)

	dlGroupChan = make(chan *downloader.Group, 1)
}

func main() {
	defer crashDetect()

	Init()

	dataFilepath := filepath.Join(dataDirectory, *groupNamePtr)
	symbols := strings.Split(*symbolCSVList, ",")

	// adapted from https://github.com/353words/stocks/blob/main/index.html
	fsSub, err := fs.Sub(staticFS, "assets")
	if err != nil {
		logh.Map[appName].Printf(logh.Error, "error calling fs.Sub: %+v", err)
	}
	http.Handle("/", http.FileServer(http.FS(fsSub)))
	http.HandleFunc("/plotly", wrappedPlotlyHandler(dlGroupChan))
	http.HandleFunc("/downloadData", wrappedDownloadYahooData(dataFilepath, symbols, dlGroupChan))
	http.HandleFunc("/symbols", wrappedSymbols(symbols))

	// Download data and put it in dlGroupChan
	downloadYahooData(*liveDataPtr, dataFilepath, symbols, dlGroupChan)

	if *runRangePtr {
		runRange()
		logh.Map[appName].Println(logh.Info, "runRange complete...")
		os.Exit(0)
	}

	// Fire the handler once to run the data. This is just so the lof file has the
	// latest trade information.
	target := fmt.Sprintf("/plotly?symbol=%s&maLength=%d&maSplit=%f", symbols[0], defs.MALengthDefault, defs.MASplitDefault)
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	wrappedPlotlyHandler(dlGroupChan)(w, req)
	// Download again (livedata is false, so this is loading the data downloaded above from file)
	// as the above call consumed the data from the channel and the registered
	// handler will not have data without calling downloadYahooData again.
	downloadYahooData(false, dataFilepath, symbols, dlGroupChan)

	logh.Map[appName].Println(logh.Info, "******************************************************")
	logh.Map[appName].Println(logh.Info, "GUI running, open a browser to http://localhost"+defs.GUIPort+",  CTRL-C to stop")
	logh.Map[appName].Println(logh.Info, "******************************************************\n\n")
	if err := http.ListenAndServe(defs.GUIPort, nil); err != nil {
		log.Fatal(err)
	}

	// Should not get here unless the server has an error.
	err = logh.ShutdownAll()
	if err != nil {
		logh.Map[appName].Printf(logh.Error, fmt.Sprintf("%#v", err))
	}
}

func downloadYahooData(liveData bool, dataFilepath string, symbols []string, dlGroupChan chan *downloader.Group) error {
	logh.Map[appName].Printf(logh.Info, "Downloading these symbols: %+v", symbols)
	group, err := financeYahoo.NewGroup(liveData, dataFilepath, *groupNamePtr, symbols)
	logh.Map[appName].Println(logh.Info, "Downloading complete")
	dlGroupChan <- group
	if err != nil {
		logh.Map[appName].Printf(logh.Error, "error calling NewGroup: %+v", err)
		return err
	}
	return nil
}

// plotlyJSON writes plot data as JSON into w
func plotlyJSON(qIssue quant.Issue, w io.Writer) error {
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
			{
				"x":    qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":    qIssue.QuantsetAsColumns.PriceMALow,
				"name": "MALow",
				"type": "scatter",
				// "stackgroup": "one",
				"line": map[string]interface{}{
					"color": "rgba(255,65,54,0.5)",
				},
				// "xaxis": "x2",
			},
			{
				"x":         qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":         qIssue.QuantsetAsColumns.PriceMA,
				"name":      "MA",
				"type":      "scatter",
				"fill":      "tonexty",
				"fillcolor": "rgba(255,164,157,0.3)",
				"line": map[string]interface{}{
					"color": "rgba(255,65,54,0.5)",
				},
				// "xaxis": "x2",
			},
			{
				"x":         qIssue.DownloaderIssue.DatasetAsColumns.Date,
				"y":         qIssue.QuantsetAsColumns.PriceMAHigh,
				"name":      "MAHigh",
				"type":      "scatter",
				"fill":      "tonexty",
				"fillcolor": "rgba(0, 140, 8, 0.3)",
				"line": map[string]interface{}{
					"color": "rgba(0, 140, 8, 0.5)", // 0, 147, 82 | 0, 139, 147 | 0, 139, 147 | 0, 65, 147 |8, 0, 147| 82, 0, 147
				},
				// "xaxis": "x2",
			},
			// Second chart
			{
				"x": qIssue.DownloaderIssue.DatasetAsColumns.Date,
				// "y":     qIssue.DownloaderIssue.DatasetAsColumns.Volume,
				// "name":  "Volume",
				"y":     qIssue.QuantsetAsColumns.TradeResults.TradeMA,
				"name":  "TradeMA",
				"type":  "scatter",
				"yaxis": "y2",
				"line": map[string]interface{}{
					"color": "rgba(255, 172, 47,1.0)",
				},
			},
			{
				"x": qIssue.DownloaderIssue.DatasetAsColumns.Date,
				// "y":     qIssue.DownloaderIssue.DatasetAsColumns.Volume,
				// "name":  "Volume",
				"y":    qIssue.QuantsetAsColumns.TradeResults.TradeGainVsTime,
				"name": "TradeGainVsTime",
				"type": "scatter",
				// "yaxis": "y3",
				"line": map[string]interface{}{
					"color": "rgba(0, 139, 147,1.0)",
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
				"title":      "Price ($ - normalized), Trade Gain",
				"autorange":  true,
				"fixedrange": false,
				"type":       "log",
			},
			"yaxis2": map[string]interface{}{
				"title":     fmt.Sprintf("Trade (algorithm:moving average, buy=%d, sell=%d)", quant.Buy, quant.Sell),
				"autorange": false,
				"range":     []float64{0.0, 1.0},
				"tick0":     0,
				"dtick":     1.0,
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
		"text": qIssue.QuantsetAsColumns.TradeResults.TradeHistory,
	}

	return json.NewEncoder(w).Encode(reply)
}

func runRange() {
	dlGroup := <-dlGroupChan
	// maLength := []int{200}
	// maSplit := []float64{0.05}
	maLength := []int{50, 60, 70, 80, 90, 100, 120, 140, 150, 160, 180, 200, 220, 240, 260, 280, 300, 350, 400, 450, 500, 600, 700, 800, 900, 1000, 1200}
	maSplit := []float64{0.015, 0.02, 0.025, 0.03, 0.04, 0.05, 0.06, 0.07, 0.08, 0.09, 0.10, 0.12, 0.14, 0.16}
	results := make([][]float64, len(maLength))
	for i := range maLength {
		splitResults := make([]float64, len(maSplit))
		splitResults[0] = 1.0
		for j := range maSplit {
			symbolResults := 1.0
			qg := quant.GetGroup(dlGroup, maLength[i], maSplit[j])
			for _, iss := range qg.Issues {
				symbolResults *= iss.QuantsetAsColumns.TradeResults.AnnualizedGain
			}
			splitResults[j] = symbolResults
		}
		results[i] = splitResults
	}

	logh.Map[appName].Printf(logh.Info, "runRange output result is product of all symbol AnnualizedGain values")
	logh.Map[appName].Printf(logh.Info, fmt.Sprintf("maSplit: %+v\n", maSplit))
	for i := range results {
		logh.Map[appName].Printf(logh.Info, fmt.Sprintf("maLength: %d %+v\n", maLength[i], results[i]))
	}
}

func wrappedDownloadYahooData(dataFilepath string, symbols []string, dlGroupChan chan *downloader.Group) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := downloadYahooData(true, dataFilepath, symbols, dlGroupChan)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func wrappedSymbols(symbols []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(symbols)
	}
}

func wrappedPlotlyHandler(dlGroupChan chan *downloader.Group) http.HandlerFunc {
	var dlGroup *downloader.Group
	var qGroup *quant.Group
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		mal := r.URL.Query().Get("maLength")
		maLength, err := strconv.Atoi(mal)
		if err != nil {
			logh.Map[appName].Printf(logh.Error, "error converting maLength value '%s' to int", mal)
			return
		}
		mas := r.URL.Query().Get("maSplit")
		maSplit, err := strconv.ParseFloat(mas, 64)
		if err != nil {
			logh.Map[appName].Printf(logh.Error, "error converting maSplit value '%s' to float", mas)
			return
		}

		select {
		case dlGroup = <-dlGroupChan:
			qGroup = quant.GetGroup(dlGroup, maLength, maSplit)
		default:
		}
		symbolIndex := -1
		for i, v := range dlGroup.Issues {
			if strings.EqualFold(v.Symbol, symbol) {
				symbolIndex = i
				break
			}
		}
		if symbolIndex == -1 {
			logh.Map[appName].Printf(logh.Warning, "Symbol %s was not found in existing data. Re-run with this symbol in the symbolCSVList.", symbol)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		qGroup.UpdateIssue(symbolIndex, maLength, maSplit)
		if err := plotlyJSON(qGroup.Issues[symbolIndex], w); err != nil {
			log.Printf("issue: %s", err)
		}
	}
}
