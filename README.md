# go-quantstudio
go-quantstudio is a GO (GOLANG) application for quantitative analysis using GO. There are two primary pieces of go-quantstudio: the downloader, and the quantitative analysis features.

Downloader highlights:
* Downloads security price data from Yahoo.
* Can be used strictly to download data and save as CSV format for use in other applications.
  * Data will always be returned in Date ascending order.
* Can be used as a package to download data, or use previously downloaded data, for use programmatically. See ExampleNewGroup() in ./downloader/financeYahoo/downloader_test.go for how to load data from a file into a Group object for programmatic use.

Quantitative analysis highlights:
* Work in progress....
* After the download, or loading previously downloaded data, an http server is used so you can browse the results graphically.
  * Supports zoom, hover tips, etc. 

```
 % go build && ./go-quantstudio --help 
Usage of ./go-quantstudio:
  -groupname string
    	Name for this group of symbols. Used for naming output files when processing groups of symbols. I.E. maybe you want to download/analyze stocks separately from ETFs (default "ETFs")
  -livedata
    	Get live data; otherwise load from file created during prior call. (default true)
  -logfile string
    	Name of log file in /Users/pauldunn/tmp/go-quantstudio; blank to print logs to terminal. (default "log.txt")
  -loglevel int
    	Logging level; default 1. Zero based index into: [debug info warning audit error] (default 1)
  -startgui
    	Runs a GUI for interacting with the data; open a browser to http://localhost:8080/ (default true)
  -symbolCSVList string
    	Comma separated list of symbols for which to download prices (default "dia,spy,qqq")
```