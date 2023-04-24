package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/paulfdunn/go-quantstudio/defs"
	"github.com/paulfdunn/go-quantstudio/downloader"
	"github.com/paulfdunn/go-quantstudio/downloader/financeYahoo"
	"github.com/paulfdunn/go-quantstudio/quant"
	"github.com/paulfdunn/go-quantstudio/quant/quantMA"
	"github.com/paulfdunn/logh"
)

var (
	// Only needed to shorten the log statements
	appName = defs.AppName

	// CLI flags
	liveDataPtr, runMARangePtr              *bool
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
	runMARangePtr = flag.Bool("runMArange", false, "When true, runs a range of moving average parameters and exits.")
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
	quantMA.Init(appName)

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
	http.HandleFunc("/plotly", quantMA.WrappedPlotlyHandlerMA(dlGroupChan))
	http.HandleFunc("/downloadData", wrappedDownloadYahooData(dataFilepath, symbols, dlGroupChan))
	http.HandleFunc("/symbols", wrappedSymbols(symbols))

	// Download data and put it in dlGroupChan
	downloadYahooData(*liveDataPtr, dataFilepath, symbols, dlGroupChan)

	if *runMARangePtr {
		runMARange()
		logh.Map[appName].Println(logh.Info, "runMARange complete...")
		os.Exit(0)
	}

	// Fire the handler once to run the data. This is just so the log file has the
	// latest trade information.
	target := fmt.Sprintf("/plotly?symbol=%s&maLength=%d&maSplit=%f", symbols[0], defs.MALengthDefault, defs.MASplitDefault)
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	quantMA.WrappedPlotlyHandlerMA(dlGroupChan)(w, req)
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

func runMARange() {
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
			qg := quantMA.GetGroup(dlGroup, maLength[i], maSplit[j])
			for _, iss := range qg.Issues {
				symbolResults *= iss.QuantsetAsColumns.Results.AnnualizedGain
			}
			splitResults[j] = symbolResults
		}
		results[i] = splitResults
	}

	logh.Map[appName].Printf(logh.Info, "runMARange output result is product of all symbol AnnualizedGain values")
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
