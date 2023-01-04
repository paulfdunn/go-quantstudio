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
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/paulfdunn/go-quantstudio/downloader"
	"github.com/paulfdunn/go-quantstudio/downloader/financeYahoo"
	"github.com/paulfdunn/go-quantstudio/quant"
	"github.com/paulfdunn/logh"
)

const (
	appName = "go-quantstudio"
	guiPort = ":8080"
)

var (
	// CLI flags
	liveDataPtr, startGUIPtr                *bool
	groupNamePtr, logFilePtr, symbolCSVList *string
	logLevel                                *int

	// dataDirectorySuffix is appended to the users home directory.
	dataDirectorySuffix = filepath.Join(`tmp`, appName)
	dataDirectory       string
)

var (
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
	liveDataPtr = flag.Bool("livedata", true, "Get live data; otherwise load from file created during prior call.")
	logFilePtr = flag.String("logfile", "log.txt", "Name of log file in "+dataDirectory+"; blank to print logs to terminal.")
	logLevel = flag.Int("loglevel", int(logh.Info), fmt.Sprintf("Logging level; default %d. Zero based index into: %v",
		int(logh.Info), logh.DefaultLevels))
	startGUIPtr = flag.Bool("startgui", true, "Runs a GUI for interacting with the data; open a browser to http://localhost"+guiPort+"/")
	symbolCSVList = flag.String("symbolCSVList", "dia,spy,qqq", "Comma separated list of symbols for which to download prices")
	flag.Parse()

	var logFilepath string
	if *logFilePtr != "" {
		logFilepath = filepath.Join(dataDirectory, *logFilePtr)
	}
	logh.New(appName, logFilepath, logh.DefaultLevels, logh.LoghLevel(*logLevel),
		logh.DefaultFlags, 100, int64(10e6))

	logh.Map[appName].Printf(logh.Debug, "user.Current(): %+v", usr)

	downloader.Init(appName)
	quant.Init(appName)
}

func main() {
	defer crashDetect()

	Init()

	dataFilepath := filepath.Join(dataDirectory, *groupNamePtr)
	symbols := strings.Split(*symbolCSVList, ",")
	logh.Map[appName].Printf(logh.Info, "Processing these symbols: %+v", symbols)
	group, err := financeYahoo.NewGroup(*liveDataPtr, dataFilepath, *groupNamePtr, symbols)
	if err != nil {
		logh.Map[appName].Printf(logh.Error, "error calling NewGroup: %+v", err)
		return
	}

	// maLength := []int{80, 100, 120, 140, 150, 160, 180, 200, 220, 240, 260}
	maLength := []int{200}
	maSplit := 0.05
	var qGroup *quant.Group
	for i := range maLength {
		qGroup = quant.Run(group, maLength[i], maSplit)
	}

	// adapted from https://github.com/353words/stocks/blob/main/index.html
	fsSub, err := fs.Sub(staticFS, "assets")
	if err != nil {
		logh.Map[appName].Printf(logh.Error, "error calling fs.Sub: %+v", err)
	}
	http.Handle("/", http.FileServer(http.FS(fsSub)))
	http.HandleFunc("/plotly", wrappedPlotlyHandler(qGroup))

	if *startGUIPtr {
		logh.Map[appName].Println(logh.Info, "GUI running, open a browser to http://localhost"+guiPort+",  CTRL-C to stop")
		if err := http.ListenAndServe(guiPort, nil); err != nil {
			log.Fatal(err)
		}
	}

	err = logh.ShutdownAll()
	if err != nil {
		logh.Map[appName].Printf(logh.Error, fmt.Sprintf("%#v", err))
	}
}

func wrappedPlotlyHandler(qGroup *quant.Group) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		symbolIndex := -1
		for i, v := range qGroup.Issues {
			if strings.EqualFold(v.DownloaderIssue.Symbol, symbol) {
				symbolIndex = i
				break
			}
		}
		if symbolIndex == -1 {
			logh.Map[appName].Printf(logh.Warning, "Symbol %s was not found in existing data. Re-run with this symbol in the symbolCSVList.", symbol)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := plotlyJSON(qGroup.Issues[symbolIndex], w); err != nil {
			log.Printf("issue: %s", err)
		}
	}
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
				"high":  qIssue.DownloaderIssue.DatasetAsColumns.AdjHigh,
				"low":   qIssue.DownloaderIssue.DatasetAsColumns.AdjLow,
				"open":  qIssue.DownloaderIssue.DatasetAsColumns.AdjOpen,
				"close": qIssue.DownloaderIssue.DatasetAsColumns.AdjClose,
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
				"y":     qIssue.QuantsetAsColumns.TradeMA,
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
				"y":     qIssue.QuantsetAsColumns.TradeGain,
				"name":  "TradeGain",
				"type":  "scatter",
				"yaxis": "y2",
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
				"rows":    2,
				"columns": 1,
			},
			"xaxis": map[string]interface{}{
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
				"autorange":  true,
				"fixedrange": false,
				"type":       "log",
			},
			"yaxis2": map[string]interface{}{
				"autorange":  true,
				"fixedrange": false,
				"type":       "log",
			},
		},
	}

	return json.NewEncoder(w).Encode(reply)
}
