package quant

import (
	"fmt"
	"math"
	"time"

	"github.com/paulfdunn/go-quantstudio/downloader"
	"github.com/paulfdunn/logh"
)

type Group struct {
	Name   string
	Issues []Issue
}

type Issue struct {
	DownloaderIssue   *downloader.Issue
	QuantsetAsColumns Quant
}

type Quant struct {
	PriceNormalizedClose []float64
	PriceNormalizedHigh  []float64
	PriceNormalizedLow   []float64
	PriceNormalizedOpen  []float64
	PriceMA              []float64
	PriceMAHigh          []float64
	PriceMALow           []float64
	TradeResults         TradeResults
}

type TradeResults struct {
	AnnualizedGain  float64
	TotalGain       float64
	TradeHistory    string
	TradeMA         []int
	TradeGainVsTime []float64
}

const (
	DateFormat = "2006-01-02"

	Buy  = 1
	Sell = 0
)

var (
	appName string

	stopLoss      = 0.9
	stopLossDelay = 15
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
	priceNormalizedClose := multiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjClose)
	priceNormalizedHigh := multiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjHigh)
	priceNormalizedLow := multiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjLow)
	priceNormalizedOpen := multiplySlice(1.0/issDAC.AdjOpen[maLength], issDAC.AdjOpen)
	priceMA := ma(maLength, true, issDAC.AdjOpen, issDAC.AdjClose)
	priceMA = multiplySlice(1.0/issDAC.AdjOpen[maLength], priceMA)
	priceMALow := multiplySlice(1.0-maSplit, priceMA)
	priceMAHigh := multiplySlice(1.0+maSplit, priceMA)
	tradeMA := tradeMA(maLength, priceNormalizedClose, priceMA, priceMAHigh, priceMALow)
	// tradeMA = tradeAddStop(tradeMA, iss)
	tradeHistory, totalGain, tradeGainVsTime := tradeGainMA(maLength, tradeMA, *iss)
	annualizedGain := annualizedGain(totalGain, issDAC.Date[0], issDAC.Date[len(issDAC.Date)-1])
	tradeResults := TradeResults{AnnualizedGain: annualizedGain, TotalGain: totalGain, TradeHistory: tradeHistory,
		TradeMA: tradeMA, TradeGainVsTime: tradeGainVsTime}
	grp.Issues[index] = Issue{DownloaderIssue: iss,
		QuantsetAsColumns: Quant{PriceNormalizedClose: priceNormalizedClose,
			PriceNormalizedHigh: priceNormalizedHigh, PriceNormalizedLow: priceNormalizedLow,
			PriceNormalizedOpen: priceNormalizedOpen,
			PriceMA:             priceMA, PriceMAHigh: priceMAHigh, PriceMALow: priceMALow,
			TradeResults: tradeResults}}
}

func annualizedGain(totalGain float64, startDate time.Time, endDate time.Time) float64 {
	diff := endDate.Sub(startDate)
	years := diff.Hours() / (24 * 365)
	return math.Pow(totalGain, 1/years)
}

// MA is the moving average of the dataSlices.
// If biasStart==true the initial points of the series are filled with the value
// of the MA on the first point after length points.
func ma(length int, biasStart bool, dataSlices ...[]float64) []float64 {
	if err := slicesAreEqualLength(dataSlices...); err != nil {
		return nil
	}

	summedData := sumSlices(dataSlices...)
	slices := float64(len(dataSlices))
	firstFullCycle := 0.0
	sum := 0.0
	dataPoints := len(dataSlices[0])
	out := make([]float64, dataPoints)
	for i := range summedData {
		sum += summedData[i] / slices
		if i >= length {
			sum -= summedData[i-length] / slices
		}
		out[i] = sum / float64(length)
		if i == length {
			firstFullCycle = out[i]
		}
	}

	if biasStart {
		for i := 0; i < length; i++ {
			out[i] = firstFullCycle
		}
	}

	return out
}

func multiplySlice(scale float64, dataSlice []float64) []float64 {
	out := make([]float64, len(dataSlice))
	for i := range dataSlice {
		out[i] = scale * dataSlice[i]
	}
	return out
}

func multiplySlices(dataSlices ...[]float64) []float64 {
	if err := slicesAreEqualLength(dataSlices...); err != nil {
		return nil
	}

	slices := len(dataSlices)
	dataPoints := len(dataSlices[0])
	out := make([]float64, dataPoints)
	for sliceIndex := 0; sliceIndex < slices; sliceIndex++ {
		for dataIndex := 0; dataIndex < dataPoints; dataIndex++ {
			if sliceIndex == 0 {
				out[dataIndex] = 1.0
			}
			out[dataIndex] *= dataSlices[sliceIndex][dataIndex]
		}
	}
	return out
}

func slicesAreEqualLength(dataSlices ...[]float64) error {
	slices := len(dataSlices)
	for i := 1; i < slices; i++ {
		if len(dataSlices[i-1]) != len(dataSlices[i]) {
			err := fmt.Errorf("slices are different lengths, processing stopped")
			logh.Map[appName].Printf(logh.Error, "%+v", err)
			return err
		}
	}
	return nil
}

func sumSlices(dataSlices ...[]float64) []float64 {
	if err := slicesAreEqualLength(dataSlices...); err != nil {
		return nil
	}

	slices := len(dataSlices)
	dataPoints := len(dataSlices[0])
	out := make([]float64, dataPoints)
	for sliceIndex := 0; sliceIndex < slices; sliceIndex++ {
		for dataIndex := 0; dataIndex < dataPoints; dataIndex++ {
			out[dataIndex] += dataSlices[sliceIndex][dataIndex]
		}
	}
	return out
}

func tradeMA(maLength int, close, priceMA, priceMAHigh, priceMALow []float64) []int {
	if err := slicesAreEqualLength(close, priceMA); err != nil {
		return nil
	}

	out := make([]int, len(priceMA))
	for i := range priceMA {
		if i <= maLength {
			out[i] = Sell
			continue
		}

		switch {
		case close[i] > priceMAHigh[i]:
			out[i] = Buy
		case close[i] < priceMALow[i]:
			out[i] = Sell
		default:
			out[i] = out[i-1]
		}
	}
	return out
}

// tradeAddStop modifies the input trade signal to sell on stop.
func tradeAddStop(trade []int, dlIssue downloader.Issue) (tradeOut []int) {
	tradeOut = make([]int, len(trade))

	highCloseSinceBuy := 0.0
	stopTriggered := false
	stopTriggeredIndex := 0
	for i := 1; i < len(trade); i++ {
		if stopTriggered {
			tradeOut[i] = Sell
			highCloseSinceBuy = dlIssue.DatasetAsColumns.AdjClose[i]
			if i > stopTriggeredIndex+stopLossDelay {
				stopTriggered = false
			}
			continue
		}

		switch {
		case trade[i-1] == Buy && trade[i] == Buy:
			if dlIssue.DatasetAsColumns.AdjClose[i] > highCloseSinceBuy {
				highCloseSinceBuy = dlIssue.DatasetAsColumns.AdjClose[i]
			}
		case trade[i-1] == Sell && trade[i] == Buy:
			if i == len(trade)-1 {
				highCloseSinceBuy = dlIssue.DatasetAsColumns.AdjClose[i]
			} else {
				highCloseSinceBuy = dlIssue.DatasetAsColumns.AdjOpen[i+1]
			}
		}

		tradeOut[i] = trade[i]
		if trade[i] == Buy && dlIssue.DatasetAsColumns.AdjClose[i]/highCloseSinceBuy < stopLoss {
			stopTriggered = true
			stopTriggeredIndex = i
			tradeOut[i] = Sell
		}
	}

	return tradeOut
}

func tradeGainMA(maLength int, trade []int, dlIssue downloader.Issue) (tradeHistory string, gain float64, tradeGain []float64) {
	seriesLen := len(dlIssue.DatasetAsColumns.AdjOpen)
	tradeGain = make([]float64, seriesLen)
	gain = 1.0
	var buyPrice float64
	var textOut string
	fd := dlIssue.DatasetAsColumns.Date[0].Format(DateFormat)
	ld := dlIssue.DatasetAsColumns.Date[len(dlIssue.DatasetAsColumns.Date)-1].Format(DateFormat)
	tradeHistory = fmt.Sprintf("first trading day: %s, last trading day: %s\n", fd, ld)
	logh.Map[appName].Printf(logh.Info, "%s", tradeHistory)
	for i := 0; i < seriesLen; i++ {
		if i <= maLength {
			tradeGain[i] = 1
			continue
		}

		switch {
		case trade[i-1] == Sell && trade[i] == Buy:
			tradeGain[i] = tradeGain[i-1]
			if i == seriesLen-1 {
				textOut = fmt.Sprintf("symbol: %s, date: %s, buyPrice: %8.2f **** TRADE TOMORROW ****",
					dlIssue.Symbol, dlIssue.DatasetAsColumns.Date[i].Format(DateFormat), buyPrice)
				logh.Map[appName].Printf(logh.Info, "%s", textOut)
				break
			}
			buyPrice = dlIssue.DatasetAsColumns.AdjOpen[i+1]
			textOut = fmt.Sprintf("symbol: %s, date: %s, buyPrice: %8.2f, ",
				dlIssue.Symbol, dlIssue.DatasetAsColumns.Date[i].Format(DateFormat), buyPrice)
		case trade[i-1] == Buy && trade[i] == Buy:
			tradeGain[i] = tradeGain[i-1] * dlIssue.DatasetAsColumns.AdjClose[i] / dlIssue.DatasetAsColumns.AdjClose[i-1]
			if i == seriesLen-1 {
				textOut += fmt.Sprintf("date: %s, ", dlIssue.DatasetAsColumns.Date[i].Format(DateFormat))
				thisGain := dlIssue.DatasetAsColumns.AdjClose[i] / buyPrice
				gain *= thisGain
				textOut += fmt.Sprintf("sellPrice: %8.2f, gain: %8.2f (TRADE STILL OPEN)", dlIssue.DatasetAsColumns.AdjClose[i], thisGain)
				tradeHistory += fmt.Sprintf("%s\n", textOut)
				logh.Map[appName].Printf(logh.Info, "%s", textOut)
			}
		case (trade[i-1] == Buy && trade[i] == Sell):
			tradeGain[i] = tradeGain[i-1] * dlIssue.DatasetAsColumns.AdjClose[i] / dlIssue.DatasetAsColumns.AdjClose[i-1]
			textOut += fmt.Sprintf("date: %s, ", dlIssue.DatasetAsColumns.Date[i].Format(DateFormat))
			if i == seriesLen-1 {
				thisGain := dlIssue.DatasetAsColumns.AdjClose[i] / buyPrice
				gain *= thisGain
				textOut += fmt.Sprintf("sellPrice: %8.2f, gain: %8.2f **** TRADE TOMORROW ****", dlIssue.DatasetAsColumns.AdjClose[i], thisGain)
				tradeHistory += fmt.Sprintf("%s\n", textOut)
				logh.Map[appName].Printf(logh.Info, "%s", textOut)
				break
			}
			tradeGain[i] = tradeGain[i] * dlIssue.DatasetAsColumns.AdjOpen[i+1] / dlIssue.DatasetAsColumns.AdjClose[i]
			thisGain := dlIssue.DatasetAsColumns.AdjOpen[i+1] / buyPrice
			gain *= thisGain
			textOut += fmt.Sprintf("sellPrice: %8.2f, gain: %8.2f", dlIssue.DatasetAsColumns.AdjOpen[i+1], thisGain)
			tradeHistory += fmt.Sprintf("%s\n", textOut)
			logh.Map[appName].Printf(logh.Info, "%s", textOut)
			textOut = ""
		case trade[i-1] == Sell && trade[i] == Sell:
			tradeGain[i] = tradeGain[i-1]
		}
	}

	start := dlIssue.DatasetAsColumns.Date[maLength]
	end := dlIssue.DatasetAsColumns.Date[seriesLen-1]
	bhGain := dlIssue.DatasetAsColumns.AdjClose[seriesLen-1] / dlIssue.DatasetAsColumns.AdjOpen[maLength]
	textOut = fmt.Sprintf("symbol: %s, buy/hold gain (annualized): %5.2f (%5.2f)",
		dlIssue.Symbol, bhGain, annualizedGain(bhGain, start, end))
	tradeHistory += fmt.Sprintf("%s\n", textOut)
	logh.Map[appName].Printf(logh.Info, textOut)
	textOut = fmt.Sprintf("symbol: %s, total gain (annualized):    %5.2f (%5.2f)\n\n",
		dlIssue.Symbol, gain, annualizedGain(gain, start, end))
	tradeHistory += fmt.Sprintf("%s\n", textOut)
	logh.Map[appName].Printf(logh.Info, textOut)

	return tradeHistory, gain, tradeGain
}
