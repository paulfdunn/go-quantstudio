package quantMA

import (
	"github.com/paulfdunn/go-quantstudio/downloader"
	"github.com/paulfdunn/go-quantstudio/quant"
	"github.com/paulfdunn/logh"
)

type Group struct {
	Name   string
	Issues []Issue
}

type Issue struct {
	DownloaderIssue   *downloader.Issue
	QuantsetAsColumns QuantMA
}

type QuantMA struct {
	PriceNormalizedClose []float64
	PriceNormalizedHigh  []float64
	PriceNormalizedLow   []float64
	PriceNormalizedOpen  []float64
	PriceMA              []float64
	PriceMAHigh          []float64
	PriceMALow           []float64
	Results              quant.Results
}

var (
	appName string
)

func Init(appNameInit string) {
	appName = appNameInit
}

func GetGroup(downloaderGroup *downloader.Group, maLength int, maSplit float64) *Group {
	logh.Map[appName].Printf(logh.Info, "calling quant.Run with maLength: %d, maSplit: %5.2f", maLength, maSplit)
	group := Group{Name: downloaderGroup.Name}
	group.Issues = make([]Issue, len(downloaderGroup.Issues))

	for index := range downloaderGroup.Issues {
		// Dont use the looping variable in a "i,v" style for loop as
		// the variable is pointing to a pointer
		group.Issues[index] = Issue{DownloaderIssue: &downloaderGroup.Issues[index]}
		group.UpdateIssue(index, maLength, maSplit)
	}

	return &group
}

func (grp *Group) UpdateIssue(index int, maLength int, maSplit float64) {
	iss := grp.Issues[index].DownloaderIssue
	issDAC := iss.DatasetAsColumns
	priceNormalizedClose := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjClose)
	priceNormalizedHigh := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjHigh)
	priceNormalizedLow := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjLow)
	priceNormalizedOpen := quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjOpen)
	priceMA := quant.MA(maLength, true, issDAC.AdjOpen, issDAC.AdjClose)
	priceMA = quant.MultiplySlice(1.0/issDAC.AdjOpen[maLength], priceMA)
	priceMALow := quant.MultiplySlice(1.0-maSplit, priceMA)
	priceMAHigh := quant.MultiplySlice(1.0+maSplit, priceMA)
	tradeMA := quant.Trade(maLength, priceNormalizedClose, priceMA, priceMAHigh, priceMALow)
	tradeHistory, totalGain, tradeGainVsTime := quant.TradeGain(maLength, tradeMA, *iss)
	annualizedGain := quant.AnnualizedGain(totalGain, issDAC.Date[0], issDAC.Date[len(issDAC.Date)-1])
	results := quant.Results{AnnualizedGain: annualizedGain, TotalGain: totalGain, TradeHistory: tradeHistory,
		TradeMA: tradeMA, TradeGainVsTime: tradeGainVsTime}
	grp.Issues[index] = Issue{DownloaderIssue: iss,
		QuantsetAsColumns: QuantMA{PriceNormalizedClose: priceNormalizedClose,
			PriceNormalizedHigh: priceNormalizedHigh, PriceNormalizedLow: priceNormalizedLow,
			PriceNormalizedOpen: priceNormalizedOpen,
			PriceMA:             priceMA, PriceMAHigh: priceMAHigh, PriceMALow: priceMALow,
			Results: results}}
}
