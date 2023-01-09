package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/paulfdunn/go-quantstudio/defs"
	"github.com/paulfdunn/logh"
)

const (
	appName = "automator"
)

var (
	// CLI flags
	logFilePtr, symbolCSVList *string
	logLevel                  *int

	// dataDirectorySuffix is appended to the users home directory.
	dataDirectorySuffix = filepath.Join(`tmp`, defs.AppName, appName)
	dataDirectory       string
)

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
	logFilePtr = flag.String("logfile", "log-chromedp.txt", "Name of log file in "+dataDirectory+"; blank to print logs to terminal.")
	logLevel = flag.Int("loglevel", int(logh.Info), fmt.Sprintf("Logging level; default %d. Zero based index into: %v",
		int(logh.Info), logh.DefaultLevels))
	symbolCSVList = flag.String("symbolCSVList", defs.SymbolsDefault, "Comma separated list of symbols for which to download prices")
	flag.Parse()

	var logFilepath string
	if *logFilePtr != "" {
		logFilepath = filepath.Join(dataDirectory, *logFilePtr)
	}
	logh.New(appName, logFilepath, logh.DefaultLevels, logh.LoghLevel(*logLevel),
		logh.DefaultFlags, 100, int64(10e6))

	logh.Map[appName].Printf(logh.Debug, "user.Current(): %+v", usr)
}

func main() {
	defer crashDetect()

	Init()

	symbols := strings.Split(*symbolCSVList, ",")
	screenShotUrl := fmt.Sprintf("http://%s/", "localhost:8080")

	getChromedpScreenShot(screenShotUrl, dataDirectory, symbols, 100)
}

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

func getChromedpScreenShot(screenShotUrl string, dataDirectory string, symbols []string, quality int) {
	// byte slice to hold captured image in bytes
	var buf []byte

	// setting image file extension to png but
	var ext string = "png"
	// if image quality is less than 100 file extension is jpeg
	if quality < 100 {
		ext = "jpeg"
	}

	logh.Map[appName].Printf(logh.Info, "Making request for screenshot using %s", screenShotUrl)

	// setting options for headless chrome to execute with
	var options []chromedp.ExecAllocatorOption
	options = append(options, chromedp.WindowSize(1400, 900))
	options = append(options, chromedp.DefaultExecAllocatorOptions[:]...)

	clickDownloadData(screenShotUrl, options)

	// setup context with options
	actx, acancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer acancel()
	// create context
	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()

	for {
		for _, symbol := range symbols {

			logh.Map[appName].Printf(logh.Info, "getting screenshot for symbol:%s", symbol)
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
			if err := ioutil.WriteFile(filepath.Join(dataDirectory, filename), buf, 0644); err != nil {
				log.Fatal(err)
			}

			//log completion and file name to
			logh.Map[appName].Printf(logh.Info, "Saved screenshot to file %s", filename)
		}

		waitForNextMarketClose()
		clickDownloadData(screenShotUrl, options)
	}
}

func clickDownloadData(screenShotUrl string, options []func(*chromedp.ExecAllocator)) {
	// setup context with options
	actx, acancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer acancel()
	// create context
	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()

	logh.Map[appName].Printf(logh.Info, "Downloading data.")
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

	statusUpdateRate := time.Hour * 8
	lastStatus := time.Now().Add(-2 * statusUpdateRate)
	for {
		if time.Now().After(nextAfternoon) {
			logh.Map[appName].Printf(logh.Info, "Time was after %+v, continuing.", nextAfternoon)
			break
		}

		// Output status messages, but rate limit.
		if time.Now().After(lastStatus.Add(statusUpdateRate)) {
			logh.Map[appName].Printf(logh.Info, "Waiting for %+v", nextAfternoon)
			lastStatus = time.Now()
		}

		time.Sleep(1 * time.Minute)
	}
}
