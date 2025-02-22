package quant

import (
	"fmt"
	"math"
	"time"

	"github.com/paulfdunn/go-helper/logh/v2"
	"github.com/paulfdunn/go-quantstudio/downloader"
)

type Results struct {
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
	lp      func(level logh.LoghLevel, v ...interface{})
	lpf     func(level logh.LoghLevel, format string, v ...interface{})

	stopLoss      = 0.9
	stopLossDelay = 15
)

func Init(appNameInit string) {
	appName = appNameInit
	lp = logh.Map[appName].Println
	lpf = logh.Map[appName].Printf
}

// AnnualizedGain will return the annualized gain given a totalGain achieved between the startDate
// and endDate.
func AnnualizedGain(totalGain float64, startDate time.Time, endDate time.Time) float64 {
	diff := endDate.Sub(startDate)
	years := diff.Hours() / (24 * 365)
	return math.Pow(totalGain, 1/years)
}

// MA is the moving average of the dataSlices.
// If biasStart==true the initial points of the series are filled with the value
// of the MA on the first point after length points.
func MA(length int, biasStart bool, dataSlices ...[]float64) []float64 {
	if err := SlicesAreEqualLength(dataSlices...); err != nil {
		return nil
	}

	summedData := SumSlices(dataSlices...)
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

// multiplySlice will multiply the input slice values by the provided scale factor
// and return the resulting slice.
func MultiplySlice(scale float64, dataSlice []float64) []float64 {
	out := make([]float64, len(dataSlice))
	for i := range dataSlice {
		out[i] = scale * dataSlice[i]
	}
	return out
}

// multiplySlice will perform multiply the input slice values by the provided scale factor,
// but only when the provided gate[i] == gateValue
// and return the resulting slice.
func MultiplySliceGated(scale float64, dataSlice []float64, gate []int, gateValue int) []float64 {
	out := make([]float64, len(dataSlice))
	for i := range dataSlice {
		out[i] = dataSlice[i]
		if gate[i] == gateValue {
			out[i] *= scale
		}
	}
	return out
}

// multiplySlices will perform a pointwise product of all inputs slices and return the resulting slice.
func MultiplySlices(dataSlices ...[]float64) []float64 {
	if err := SlicesAreEqualLength(dataSlices...); err != nil {
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

// offsetSlice will add an offset to all values in the input slice and return the resulting slice.
func OffsetSlice(offset float64, dataSlice []float64) []float64 {
	out := make([]float64, len(dataSlice))
	for i := range dataSlice {
		out[i] = offset + dataSlice[i]
	}
	return out
}

// reciprocolSlice will perform a pointwise reciprical of the input slice and return the resulting slice.
func ReciprocolSlice(dataSlice []float64) []float64 {
	out := make([]float64, len(dataSlice))
	for i := range dataSlice {
		out[i] = 1 / dataSlice[i]
	}
	return out
}

// slicesAreEqualLength returns an error if the input slices are not of the same length.
func SlicesAreEqualLength(dataSlices ...[]float64) error {
	slices := len(dataSlices)
	for i := 1; i < slices; i++ {
		if len(dataSlices[i-1]) != len(dataSlices[i]) {
			err := fmt.Errorf("slices are different lengths, processing stopped")
			lpf(logh.Error, "%+v", err)
			return err
		}
	}
	return nil
}

// sumSlices will perform a pointwise sum of all inputs slices and return the resulting slice.
func SumSlices(dataSlices ...[]float64) []float64 {
	if err := SlicesAreEqualLength(dataSlices...); err != nil {
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

// Trade delays delay number of points, then compares price to the buyLevel and sellLevel,
// and returns an output slice indicating Buy or Sell at
// each point. Note that Sell is returned for the first delay number of points.
func TradeOnPrice(delay int, close, price, buyLevel, sellLevel []float64) []int {
	if err := SlicesAreEqualLength(close, price); err != nil {
		return nil
	}

	out := make([]int, len(price))
	for i := range price {
		if i <= delay-1 {
			out[i] = Sell
			continue
		}

		switch {
		case close[i] > buyLevel[i]:
			out[i] = Buy
		case close[i] < sellLevel[i]:
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

// TradeGain takes in input slice trade with values Buy/Sell, and after delay number of points,
// applies the Buy/Sell signals to dlIssue to proces a tradeHistory, gain (total gain), and
// tradeGain (accumulated gain/loss at each point).
func TradeGain(delay int, trade []int, dlIssue downloader.Issue) (tradeHistory string, gain float64, tradeGain []float64) {
	seriesLen := len(dlIssue.DatasetAsColumns.AdjOpen)
	tradeGain = make([]float64, seriesLen)
	gain = 1.0
	var buyPrice float64
	var textOut string
	fd := dlIssue.DatasetAsColumns.Date[0].Format(DateFormat)
	ld := dlIssue.DatasetAsColumns.Date[len(dlIssue.DatasetAsColumns.Date)-1].Format(DateFormat)
	tradeHistory = fmt.Sprintf("first trading day: %s, last trading day: %s\n", fd, ld)
	lpf(logh.Info, "%s", tradeHistory)
	for i := 0; i < seriesLen; i++ {
		if i <= delay-1 {
			tradeGain[i] = 1
			continue
		}

		switch {
		case trade[i-1] == Sell && trade[i] == Buy:
			tradeGain[i] = tradeGain[i-1]
			if i == seriesLen-1 {
				textOut = fmt.Sprintf("symbol: %s, date: %s, buyPrice: %8.2f **** TRADE TOMORROW ****",
					dlIssue.Symbol, dlIssue.DatasetAsColumns.Date[i].Format(DateFormat), buyPrice)
				lpf(logh.Info, "%s", textOut)
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
				lpf(logh.Info, "%s", textOut)
			}
		case (trade[i-1] == Buy && trade[i] == Sell):
			tradeGain[i] = tradeGain[i-1] * dlIssue.DatasetAsColumns.AdjClose[i] / dlIssue.DatasetAsColumns.AdjClose[i-1]
			textOut += fmt.Sprintf("date: %s, ", dlIssue.DatasetAsColumns.Date[i].Format(DateFormat))
			if i == seriesLen-1 {
				thisGain := dlIssue.DatasetAsColumns.AdjClose[i] / buyPrice
				gain *= thisGain
				textOut += fmt.Sprintf("sellPrice: %8.2f, gain: %8.2f **** TRADE TOMORROW ****", dlIssue.DatasetAsColumns.AdjClose[i], thisGain)
				tradeHistory += fmt.Sprintf("%s\n", textOut)
				lpf(logh.Info, "%s", textOut)
				break
			}
			tradeGain[i] = tradeGain[i] * dlIssue.DatasetAsColumns.AdjOpen[i+1] / dlIssue.DatasetAsColumns.AdjClose[i]
			thisGain := dlIssue.DatasetAsColumns.AdjOpen[i+1] / buyPrice
			gain *= thisGain
			textOut += fmt.Sprintf("sellPrice: %8.2f, gain: %8.2f", dlIssue.DatasetAsColumns.AdjOpen[i+1], thisGain)
			tradeHistory += fmt.Sprintf("%s\n", textOut)
			lpf(logh.Info, "%s", textOut)
			textOut = ""
		case trade[i-1] == Sell && trade[i] == Sell:
			tradeGain[i] = tradeGain[i-1]
		}
	}

	start := dlIssue.DatasetAsColumns.Date[delay]
	end := dlIssue.DatasetAsColumns.Date[seriesLen-1]
	bhGain := dlIssue.DatasetAsColumns.AdjClose[seriesLen-1] / dlIssue.DatasetAsColumns.AdjOpen[delay]
	textOut = fmt.Sprintf("symbol: %s, buy/hold gain (annualized): %5.2f (%5.2f)",
		dlIssue.Symbol, bhGain, AnnualizedGain(bhGain, start, end))
	tradeHistory += fmt.Sprintf("%s\n", textOut)
	lpf(logh.Info, textOut)
	textOut = fmt.Sprintf("symbol: %s, total gain (annualized):    %5.2f (%5.2f)\n\n",
		dlIssue.Symbol, gain, AnnualizedGain(gain, start, end))
	tradeHistory += fmt.Sprintf("%s\n", textOut)
	lpf(logh.Info, textOut)

	return tradeHistory, gain, tradeGain
}
