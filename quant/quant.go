package quant

import (
	"fmt"

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
	TradeMA              []int
	TradeGain            []float64
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

func Run(downloaderGroup *downloader.Group, maLength int, maSplit float64) *Group {
	logh.Map[appName].Printf(logh.Info, "calling quant.Run with maLength: %d, maSplit: %5.2f", maLength, maSplit)
	group := Group{Name: downloaderGroup.Name}
	group.Issues = make([]Issue, len(downloaderGroup.Issues))

	for i := range downloaderGroup.Issues {
		priceNormalizedClose := multiplySlice(1.0/downloaderGroup.Issues[i].DatasetAsColumns.AdjOpen[maLength],
			downloaderGroup.Issues[i].DatasetAsColumns.AdjClose)
		priceNormalizedHigh := multiplySlice(1.0/downloaderGroup.Issues[i].DatasetAsColumns.AdjOpen[maLength],
			downloaderGroup.Issues[i].DatasetAsColumns.AdjHigh)
		priceNormalizedLow := multiplySlice(1.0/downloaderGroup.Issues[i].DatasetAsColumns.AdjOpen[maLength],
			downloaderGroup.Issues[i].DatasetAsColumns.AdjLow)
		priceNormalizedOpen := multiplySlice(1.0/downloaderGroup.Issues[i].DatasetAsColumns.AdjOpen[maLength],
			downloaderGroup.Issues[i].DatasetAsColumns.AdjOpen)
		priceMA := ma(maLength, true, downloaderGroup.Issues[i].DatasetAsColumns.AdjOpen,
			downloaderGroup.Issues[i].DatasetAsColumns.AdjClose)
		priceMA = multiplySlice(1.0/downloaderGroup.Issues[i].DatasetAsColumns.AdjOpen[maLength],
			priceMA)
		priceMALow := multiplySlice(1.0-maSplit, priceMA)
		priceMAHigh := multiplySlice(1.0+maSplit, priceMA)
		tradeMA := tradeMA(maLength, priceNormalizedClose, priceMA, priceMAHigh, priceMALow)
		// tradeMA = tradeAddStop(tradeMA, downloaderGroup.Issues[i])
		_, tradeGain := tradeGainMA(maLength, tradeMA, downloaderGroup.Issues[i])
		group.Issues[i] = Issue{DownloaderIssue: &downloaderGroup.Issues[i],
			QuantsetAsColumns: Quant{PriceNormalizedClose: priceNormalizedClose,
				PriceNormalizedHigh: priceNormalizedHigh, PriceNormalizedLow: priceNormalizedLow,
				PriceNormalizedOpen: priceNormalizedOpen,
				PriceMA:             priceMA, PriceMAHigh: priceMAHigh, PriceMALow: priceMALow,
				TradeMA: tradeMA, TradeGain: tradeGain}}
	}

	return &group
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

func tradeGainMA(maLength int, trade []int, dlIssue downloader.Issue) (gain float64, tradeGain []float64) {
	seriesLen := len(dlIssue.DatasetAsColumns.AdjOpen)
	tradeGain = make([]float64, seriesLen)
	gain = 1.0
	var buyPrice float64
	var textOut string
	for i := 0; i < seriesLen; i++ {
		if i <= maLength {
			tradeGain[i] = 1
			continue
		}

		switch {
		case trade[i-1] == Sell && trade[i] == Buy:
			tradeGain[i] = tradeGain[i-1]
			if i == seriesLen-1 {
				logh.Map[appName].Println(logh.Info, "**** TRADE TOMOROWW ****")
				break
			}
			buyPrice = dlIssue.DatasetAsColumns.AdjOpen[i+1]
			textOut = fmt.Sprintf("date: %s, symbol: %s, buyPrice: %8.2f, ",
				dlIssue.DatasetAsColumns.Date[i].Format(DateFormat),
				dlIssue.Symbol, buyPrice)
		case trade[i-1] == Buy && trade[i] == Buy:
			tradeGain[i] = tradeGain[i-1] * dlIssue.DatasetAsColumns.AdjClose[i] / dlIssue.DatasetAsColumns.AdjClose[i-1]
		case trade[i-1] == Buy && trade[i] == Sell:
			tradeGain[i] = tradeGain[i-1] * dlIssue.DatasetAsColumns.AdjClose[i] / dlIssue.DatasetAsColumns.AdjClose[i-1]
			textOut += fmt.Sprintf("date: %s, ", dlIssue.DatasetAsColumns.Date[i].Format(DateFormat))
			if i == seriesLen-1 {
				logh.Map[appName].Println(logh.Info, "**** TRADE TOMOROWW ****")
				thisGain := dlIssue.DatasetAsColumns.AdjClose[i] / buyPrice
				gain *= thisGain
				textOut += fmt.Sprintf("sellPrice: %8.2f, gain: %8.2f", dlIssue.DatasetAsColumns.AdjClose[i], thisGain)
				logh.Map[appName].Printf(logh.Info, "%s", textOut)
				break
			}
			tradeGain[i] = tradeGain[i] * dlIssue.DatasetAsColumns.AdjOpen[i+1] / dlIssue.DatasetAsColumns.AdjClose[i]
			thisGain := dlIssue.DatasetAsColumns.AdjOpen[i+1] / buyPrice
			gain *= thisGain
			textOut += fmt.Sprintf("sellPrice: %8.2f, gain: %8.2f", dlIssue.DatasetAsColumns.AdjOpen[i+1], thisGain)
			logh.Map[appName].Printf(logh.Info, "%s", textOut)
			textOut = ""
		case trade[i-1] == Sell && trade[i] == Sell:
			tradeGain[i] = tradeGain[i-1]
		}
	}

	logh.Map[appName].Printf(logh.Info, "symbol: %s, buy/hold gain: %8.2f",
		dlIssue.Symbol, dlIssue.DatasetAsColumns.AdjClose[seriesLen-1]/dlIssue.DatasetAsColumns.AdjOpen[0])
	logh.Map[appName].Printf(logh.Info, "symbol: %s, total gain:    %8.2f\n\n", dlIssue.Symbol, gain)

	return gain, tradeGain
}
