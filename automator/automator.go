package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/paulfdunn/go-quantstudio/defs"
	"github.com/paulfdunn/logh"
)

const (
	appName = "automator"
	// Set headless false for debugging, as headless hides the browser window
	headless = true
)

var (
	// CLI flags
	logFilePtr *string
	logLevel   *int
	lp         func(level logh.LoghLevel, v ...interface{})
	lpf        func(level logh.LoghLevel, format string, v ...interface{})

	// dataDirectorySuffix is appended to the users home directory.
	dataDirectorySuffix = filepath.Join(`tmp`, defs.AppName, appName)
	dataDirectory       string
)

func Init() {
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
	logFilePtr = flag.String("logfile", "automator.log", "Name of log file in "+dataDirectory+"; blank to print logs to terminal.")
	logLevel = flag.Int("loglevel", int(logh.Info), fmt.Sprintf("Logging level; default %d. Zero based index into: %v",
		int(logh.Info), logh.DefaultLevels))
	flag.Parse()

	var logFilepath string
	if *logFilePtr != "" {
		logFilepath = filepath.Join(dataDirectory, *logFilePtr)
	}
	logh.New(appName, logFilepath, logh.DefaultLevels, logh.LoghLevel(*logLevel),
		logh.DefaultFlags, 100, int64(10e6))

	lp = logh.Map[appName].Println
	lpf = logh.Map[appName].Printf
	lpf(logh.Debug, "user.Current(): %+v", usr)
}

func main() {
	defer crashDetect()

	Init()

	// Get the symbols that are loaded in go-quantstudio, then get screen shots for all symbols
	screenShotUrl := fmt.Sprintf("http://%s/", "localhost:8080")
	symbols, err := getLoadedSymbols()
	if err != nil {
		lpf(logh.Error, "automator could not load symbols from go-quantstudio, exiting.")
		os.Exit(1)
	}

	getChromedpScreenShotsForAllSymbols(screenShotUrl, symbols, 100)
}

func clickDownloadData(screenShotUrl string) {
	// setting options for headless chrome to execute with
	var options []chromedp.ExecAllocatorOption
	options = append(options,
		append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", headless), chromedp.WindowSize(1400, 900))...)

	// setup context with options
	actx, acancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer acancel()
	// create context
	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()

	lpf(logh.Info, "Downloading data.")
	//configuring a set of tasks to be run
	tasks := chromedp.Tasks{
		//loads page of the URL
		chromedp.Navigate(screenShotUrl),
		//waits for 5 secs
		chromedp.Sleep(5 * time.Second),
		chromedp.Click(`button[id='downloadData']`, chromedp.NodeVisible),
		chromedp.Sleep(10 * time.Second),
	}

	// running the tasks configured earlier and logging any errors
	if err := chromedp.Run(ctx, tasks); err != nil {
		log.Fatal(err)
	}
}

func crashDetect() {
	if err := recover(); err != nil {
		errOut := fmt.Sprintf("panic: %+v\n%+v", err, string(debug.Stack()))
		fmt.Println(errOut)
		lp(logh.Error, errOut)
		errShutdown := logh.ShutdownAll()
		if errShutdown != nil {
			lpf(logh.Error, fmt.Sprintf("%#v", errShutdown))
		}
	}
}

func getChromedpScreenShotsForAllSymbols(screenShotUrl string, symbols []string, quality int) {
	lpf(logh.Info, "Making request for screenshot using %s", screenShotUrl)

	for {
		for _, symbol := range symbols {
			getScreenshotForSymbol(screenShotUrl, symbol, quality)
		}

		waitForNextMarketClose()
		// Debug only
		// waitUntil(time.Now().Add(time.Minute))

		clickDownloadData(screenShotUrl)
	}
}

// getLoadedSymbols gets the symbols that are running in go-quantstudio.
func getLoadedSymbols() ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost%s/symbols", defs.GUIPort))
	if err != nil {
		lpf(logh.Error, "getting symbols: %s", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		lpf(logh.Error, "getting symbol bytes: %s", err)
		return nil, err
	}
	var symbols []string
	err = json.Unmarshal(body, &symbols)
	if err != nil {
		lpf(logh.Error, "unmarshalling symbols: %s", err)
		return nil, err
	}

	return symbols, nil
}

func getScreenshotForSymbol(screenShotUrl string, symbol string, quality int) {
	// byte slice to hold captured image in bytes
	var buf []byte

	// setting image file extension to png but
	var ext string = "png"
	// if image quality is less than 100 file extension is jpeg
	if quality < 100 {
		ext = "jpeg"
	}

	// setting options for headless chrome to execute with
	var options []chromedp.ExecAllocatorOption
	options = append(options,
		append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", headless), chromedp.WindowSize(1400, 900))...)

	// setup context with options
	actx, acancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer acancel()
	// create context
	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()

	lpf(logh.Info, "getting screenshot for symbol:%s", symbol)
	//configuring a set of tasks to be run
	tasks := chromedp.Tasks{
		//loads page of the URL
		chromedp.Navigate(screenShotUrl),
		//waits for 5 secs
		chromedp.Sleep(5 * time.Second),
		chromedp.Click(`input[id='symbol']`, chromedp.NodeVisible),
		chromedp.SendKeys(`#symbol`, fmt.Sprintf("\b\b\b\b\b\b%s\n", symbol), chromedp.ByID),
		chromedp.Sleep(5 * time.Second),
		//Captures Screenshot with current window size
		chromedp.CaptureScreenshot(&buf),
		//captures full-page screenshot (uncomment to take fullpage screenshot)
		//chromedp.FullScreenshot(&buf,quality),
	}

	// running the tasks configured earlier and logging any errors
	if err := chromedp.Run(ctx, tasks); err != nil {
		log.Fatal(err)
	}
	//naming file using provided URL without "/"s and current unix datetime
	filename := fmt.Sprintf("%s.%s", symbol, ext)

	//write byte slice data of standard screenshot to file
	if err := os.WriteFile(filepath.Join(dataDirectory, filename), buf, 0644); err != nil {
		log.Fatal(err)
	}

	//log completion and file name to
	lpf(logh.Info, "Saved screenshot to file %s", filename)

}

// listenForNetworkEvent shows how to get responses.
// func listenForNetworkEvent(ctx context.Context) {
// 	chromedp.ListenTarget(ctx, func(ev interface{}) {
// 		switch ev := ev.(type) {

// 		case *network.EventResponseReceived:
// 			resp := ev.Response
// 			lpf(logh.Info, "response status: %d", resp.Status)
// 			if len(resp.Headers) != 0 {
// 				lpf(logh.Debug, "received headers: %s", resp.Headers)
// 			}
// 		}
// 	})

// }

func waitForNextMarketClose() {
	now := time.Now()
	thisAfternoon := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, time.UTC)
	var nextAfternoon time.Time
	if now.Weekday() >= time.Monday && now.Weekday() <= time.Friday && now.Before(thisAfternoon) {
		nextAfternoon = thisAfternoon
	} else if now.Weekday() == time.Friday {
		nextAfternoon = time.Date(now.Year(), now.Month(), now.Day()+3, 23, 0, 0, 0, time.UTC)
	} else if now.Weekday() == time.Saturday {
		nextAfternoon = time.Date(now.Year(), now.Month(), now.Day()+2, 23, 0, 0, 0, time.UTC)
	} else {
		nextAfternoon = time.Date(now.Year(), now.Month(), now.Day()+1, 23, 0, 0, 0, time.UTC)
	}

	statusUpdateRate := time.Hour * 4
	waitUntil(nextAfternoon, statusUpdateRate)
}

func waitUntil(nextAfternoon time.Time, statusUpdateRate time.Duration) {
	lastStatus := time.Now().Add(-2 * statusUpdateRate)
	for {
		if time.Now().After(nextAfternoon) {
			lpf(logh.Info, "Time was after %+v, continuing.", nextAfternoon)
			break
		}

		// Output status messages, but rate limit.
		if time.Now().After(lastStatus.Add(statusUpdateRate)) {
			lpf(logh.Info, "Waiting for %+v", nextAfternoon)
			lastStatus = time.Now()
		}

		time.Sleep(1 * time.Minute)
	}
}
