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
	Trade           []int
	TradeGainVsTime []float64
}

type TradeOnSignalLongRebuyInputs struct {
	DlIssue           *downloader.Issue
	AllowedLongRebuys int
	ConsecutiveUpDays int
	Stop              float64
}

const (
	DateFormat = "2006-01-02"

	LongRebuy = 2
	LongBuy   = 1
	Close     = 0
	ShortSell = -1

	// TradeGap is the minimum number of points between trades. Settlement time is 1 days
	// so 1 is used to insure another trade is not opened until the previous one is settled.
	TradeGap = 1
)

var (
	appName string
	lp      func(level logh.LoghLevel, v ...interface{})
	lpf     func(level logh.LoghLevel, format string, v ...interface{})

	// stopLoss      = 0.9
	// stopLossDelay = 15
)

func Init(appNameInit string) {
	appName = appNameInit
	lp = logh.Map[appName].Println
	lpf = logh.Map[appName].Printf
}

// Abs will return the absolute value of the input
func Abs(input []float64) []float64 {
	out := make([]float64, len(input))
	for i := 0; i < len(input); i++ {
		out[i] = math.Abs(input[i])
	}
	return out
}

// AnnualizedGain will return the annualized gain given a totalGain achieved between the startDate
// and endDate.
func AnnualizedGain(totalGain float64, startDate time.Time, endDate time.Time) float64 {
	diff := endDate.Sub(startDate)
	years := diff.Hours() / (24 * 365)
	return math.Pow(totalGain, 1/years)
}

// ConsecutiveDirection will return a slice of integers indicating the cummulative direction of the input slice.
// The output slice will be 0 if the input is equal to the previous point, +1 if the input is increasing,
// and -1 if the input is decreasing. The output slice will be 0 for the first point.
// The output slice will be 0 for the first point.
func ConsecutiveDirection(input []float64) []int {
	out := make([]int, len(input))
	for i := 1; i < len(input); i++ {
		if input[i] == input[i-1] {
			out[i] = 0
		} else if out[i-1] >= 0 && input[i] > input[i-1] {
			out[i] = out[i-1] + 1
		} else if out[i-1] <= 0 && input[i] > input[i-1] {
			out[i] = 1
		} else if out[i-1] <= 0 && input[i] < input[i-1] {
			out[i] = out[i-1] - 1
		} else if out[i-1] >= 0 && input[i] < input[i-1] {
			out[i] = -1
		}
	}
	return out
}

// Differentiate will return the backwards difference of the input slice.
// The first point is set to 0.
func Differentiate(input []float64) []float64 {
	out := make([]float64, len(input))
	out[0] = 0
	for i := 1; i < len(input); i++ {
		out[i] = input[i] - input[i-1]
	}
	return out
}

// EMA is the exponential moving average of the dataSlices.
// If biasStart==true the initial points of the series are filled with the value
// of the MA on the first point after length points.
func EMA(length int, biasStart bool, dataSlices ...[]float64) ([]float64, error) {
	if err := SlicesAreEqualLength(dataSlices...); err != nil {
		lpf(logh.Error, "%+v", err)
		return nil, err
	}

	summedData, err := SumSlices(dataSlices...)
	if err != nil {
		lpf(logh.Error, "%+v", err)
		return nil, err
	}
	slices := float64(len(dataSlices))
	firstFullCycle := 0.0
	dataPoints := len(dataSlices[0])
	out := make([]float64, dataPoints)
	weights := make([]float64, length)
	totalWeight := 0.0
	for i := 0; i < length; i++ {
		weights[i] = math.Exp(3.0 * float64(-i) / float64(length))
		totalWeight += weights[i]
	}
	for i := range summedData {
		sum := 0.0
		for j := 0; j < length; j++ {
			if i-j < 0 {
				break
			}
			sum += weights[j] * summedData[i-j] / slices
		}
		out[i] = sum / totalWeight
		if i == length {
			firstFullCycle = out[i]
		}
	}

	if biasStart {
		for i := 0; i < length; i++ {
			out[i] = firstFullCycle
		}
	}

	return out, nil
}

// MarketClosedGain will return the cummulative gain for an issue that is only held during market close.
// I.E. you are constantly buying at close, selling at open.
func MarketClosedGain(open []float64, close []float64) ([]float64, error) {
	if err := SlicesAreEqualLength(open, close); err != nil {
		lpf(logh.Error, "%+v", err)
		return nil, err
	}

	out := make([]float64, len(open))
	out[0] = 1.0
	for i := 1; i < len(open); i++ {
		out[i] = out[i-1] * (open[i] / close[i-1])
	}

	return out, nil
}

// MarketOpenGain will return the cummulative gain for an issue that is only held during market open.
// I.E. you are constantly buying at open, selling at close.
func MarketOpenGain(open []float64, close []float64) ([]float64, error) {
	if err := SlicesAreEqualLength(open, close); err != nil {
		lpf(logh.Error, "%+v", err)
		return nil, err
	}

	out := make([]float64, len(open))
	out[0] = 1.0
	for i := 1; i < len(open); i++ {
		out[i] = out[i-1] * (close[i] / open[i])
	}

	return out, nil
}

// MA is the moving average of the dataSlices.
// If biasStart==true the initial points of the series are filled with the value
// of the MA on the first point after length points.
func MA(length int, biasStart bool, dataSlices ...[]float64) ([]float64, error) {
	if err := SlicesAreEqualLength(dataSlices...); err != nil {
		lpf(logh.Error, "%+v", err)
		return nil, err
	}

	summedData, err := SumSlices(dataSlices...)
	if err != nil {
		lpf(logh.Error, "%+v", err)
		return nil, err
	}
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

	return out, nil
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
func MultiplySlices(dataSlices ...[]float64) ([]float64, error) {
	if err := SlicesAreEqualLength(dataSlices...); err != nil {
		lpf(logh.Error, "%+v", err)
		return nil, err
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
	return out, nil
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
			err := fmt.Errorf("slices are different lengths")
			return err
		}
	}
	return nil
}

// sumSlices will perform a pointwise sum of all inputs slices and return the resulting slice.
func SumSlices(dataSlices ...[]float64) ([]float64, error) {
	if err := SlicesAreEqualLength(dataSlices...); err != nil {
		lpf(logh.Error, "%+v", err)
		return nil, err
	}

	slices := len(dataSlices)
	dataPoints := len(dataSlices[0])
	out := make([]float64, dataPoints)
	for sliceIndex := 0; sliceIndex < slices; sliceIndex++ {
		for dataIndex := 0; dataIndex < dataPoints; dataIndex++ {
			out[dataIndex] += dataSlices[sliceIndex][dataIndex]
		}
	}
	return out, nil
}

// TradeGain takes in input slice trade with values [LongRebuy, LongBuy, Close, ShortSell], and after delay number of points,
// applies the [LongRebuy, LongBuy, Close, ShortSell] signals to dlIssue to proces a tradeHistory, gain (total gain), and
// tradeGain (accumulated gain/loss at each point).
// All trades MUST Close between either a LongBuy or a ShortSell.
func TradeGain(delay int, trade []int, dlIssue downloader.Issue) (tradeHistory string, gain float64, tradeGain []float64) {
	seriesLen := len(dlIssue.DatasetAsColumns.AdjOpen)
	// tradeGain is the product of all daily changes in Issue price while a trades are open. This is useful
	// for graphing the progression of gains.
	tradeGain = make([]float64, seriesLen)
	// gain is the product of all trade gains; where gain is sale_price/purchase_price.
	// This will be slightly different than tradeGain due to floating point errors.
	gain = 1.0
	var longBuyPrice, shortSellPrice float64
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
		case trade[i-1] == Close && (trade[i] >= LongBuy || trade[i] <= ShortSell):
			action := ""
			var price float64
			if i < seriesLen-1 {
				price = dlIssue.DatasetAsColumns.AdjOpen[i+1]
			} else {
				price = dlIssue.DatasetAsColumns.AdjOpen[i]
			}
			if trade[i] >= LongBuy {
				action = "long buy"
				longBuyPrice = price
			} else {
				action = "short sell"
				shortSellPrice = price
			}
			tradeGain[i] = tradeGain[i-1]
			if i == seriesLen-1 {
				action = fmt.Sprintf("**** %s TOMORROW ****", action)
			}
			tradeHistory += fmt.Sprintf("symbol: %s, date: %s, %s price: %8.2f, ",
				dlIssue.Symbol, dlIssue.DatasetAsColumns.Date[i].Format(DateFormat), action, price)
		case (trade[i-1] >= LongBuy && trade[i] >= LongBuy) || (trade[i-1] <= ShortSell && trade[i] <= ShortSell):
			action := ""
			var price, pointGain, thisGain float64
			pointGain = dlIssue.DatasetAsColumns.AdjClose[i] / dlIssue.DatasetAsColumns.AdjClose[i-1]
			if trade[i] >= LongBuy {
				action = "long sell"
				price = longBuyPrice
				thisGain = dlIssue.DatasetAsColumns.AdjClose[i] / price
			} else {
				action = "short buy"
				price = shortSellPrice
				pointGain = 1 / pointGain
				thisGain = price / dlIssue.DatasetAsColumns.AdjClose[i]
			}
			tradeGain[i] = tradeGain[i-1] * pointGain
			if i == seriesLen-1 {
				tradeHistory += fmt.Sprintf("date: %s, ", dlIssue.DatasetAsColumns.Date[i].Format(DateFormat))
				gain *= thisGain
				tradeHistory += fmt.Sprintf("%s price: %8.2f, gain: %8.2f (TRADE STILL OPEN)\n", action, dlIssue.DatasetAsColumns.AdjClose[i], thisGain)
			}
		case (trade[i-1] >= LongBuy || trade[i-1] <= ShortSell) && trade[i] == Close:
			action := ""
			var finalGain, pointGain, price, thisGain float64
			pointGain = dlIssue.DatasetAsColumns.AdjClose[i] / dlIssue.DatasetAsColumns.AdjClose[i-1]
			if i < seriesLen-1 {
				price = dlIssue.DatasetAsColumns.AdjOpen[i+1]
				finalGain = dlIssue.DatasetAsColumns.AdjOpen[i+1] / dlIssue.DatasetAsColumns.AdjClose[i]
			} else {
				price = dlIssue.DatasetAsColumns.AdjClose[i]
				// There is no final gain to calculate, trade is closing at the final point.
				finalGain = 1.0
			}
			if trade[i-1] >= LongBuy {
				action = "long sell"
				thisGain = price / longBuyPrice
				// Protect against logic errors by setting longBuyPrice to NaN when not in use.
				longBuyPrice = math.NaN()
				tradeGain[i] = tradeGain[i-1] * pointGain * finalGain
			} else {
				action = "short buy"
				thisGain = shortSellPrice / price
				// Protect against logic errors by setting shortSellPrice to NaN when not in use.
				shortSellPrice = math.NaN()
				tradeGain[i] = tradeGain[i-1] * (1 / pointGain) * (1 / finalGain)
			}
			if i == seriesLen-1 {
				action = fmt.Sprintf("**** %s TOMORROW ****", action)
			}
			tradeHistory += fmt.Sprintf("date: %s, ", dlIssue.DatasetAsColumns.Date[i].Format(DateFormat))
			gain *= thisGain
			tradeHistory += fmt.Sprintf("%s price: %8.2f, gain: %8.2f\n", action, price, thisGain)
		case trade[i-1] == Close && trade[i] == Close:
			tradeGain[i] = tradeGain[i-1]
		}
	}

	start := dlIssue.DatasetAsColumns.Date[delay]
	end := dlIssue.DatasetAsColumns.Date[seriesLen-1]
	bhGain := dlIssue.DatasetAsColumns.AdjClose[seriesLen-1] / dlIssue.DatasetAsColumns.AdjOpen[delay]
	tradeHistory += fmt.Sprintf("symbol: %s, buy/hold gain (annualized): %5.2f (%5.2f)\n",
		dlIssue.Symbol, bhGain, AnnualizedGain(bhGain, start, end))
	tradeHistory += fmt.Sprintf("symbol: %s, total gain (annualized):    %5.2f (%5.2f)\n\n",
		dlIssue.Symbol, gain, AnnualizedGain(gain, start, end))
	lpf(logh.Info, tradeHistory)

	return tradeHistory, gain, tradeGain
}

// TradeOnSignal delays delay number of points, then compares signal to the buyLevel and sellLevel,
// and returns an output slice indicating [LongRebuy, LongBuy, Close, ShortSell] at
// each point. Note that Close is returned for the first delay number of points.
// It is invalid for a both a long and short trade to be open at the same time.
// TradeOnSignalLongRebuyInputs is used to enable re-buy on long trades. This is useful for
// getting back into a trade that rapidly turns around. The only requirement is that
// consecutiveUpDays >= rebuyInputs.ConsecutiveUpDays. The only transition from LongRebuy is to
// LongBuy (the signal ultimately exceeds the longBuyLevel), or the stop is hit and the trade is closed.
func TradeOnSignal(rebuyInputs *TradeOnSignalLongRebuyInputs, delay int, signal, longBuyLevel, longSellLevel, shortSellLevel, shortBuyLevel []float64) ([]int, error) {
	// longRebuyEnabled is an attempt to get back in sooner in the event of a sudden direction change.
	longRebuyEnabled := false
	if rebuyInputs != nil && rebuyInputs.DlIssue != nil {
		longRebuyEnabled = true
	}

	if longRebuyEnabled {
		if err := SlicesAreEqualLength(signal, rebuyInputs.DlIssue.DatasetAsColumns.AdjClose, rebuyInputs.DlIssue.DatasetAsColumns.AdjOpen); err != nil {
			lpf(logh.Error, "%+v", err)
			return nil, err
		}
	}
	if err := SlicesAreEqualLength(signal, longBuyLevel, longSellLevel); err != nil {
		lpf(logh.Error, "%+v", err)
		return nil, err
	}

	consecutiveUpDays := 0
	longRebuys := 0
	seriesLen := 0
	if longRebuyEnabled {
		seriesLen = len(rebuyInputs.DlIssue.DatasetAsColumns.AdjClose)
	}
	out := make([]int, len(signal))
	out[0] = Close
	gap := 0
	longRebuyPrice := 0.0
	for i := 1; i < len(signal); i++ {
		if longRebuyEnabled && rebuyInputs.DlIssue.DatasetAsColumns.AdjClose[i] > rebuyInputs.DlIssue.DatasetAsColumns.AdjClose[i-1] {
			consecutiveUpDays++
		} else {
			consecutiveUpDays = 0
		}

		if i <= delay-1 || gap > 0 {
			out[i] = Close
			if gap > 0 {
				gap--
			}
			continue
		}

		switch {
		case !(out[i-1] == ShortSell) && longBuyLevel != nil && signal[i] > longBuyLevel[i]:
			out[i] = LongBuy
		case out[i-1] == LongBuy && longSellLevel != nil && signal[i] < longSellLevel[i]:
			out[i] = Close
			longRebuys = 0
		case longRebuyEnabled && out[i-1] == LongRebuy && rebuyInputs.DlIssue.DatasetAsColumns.AdjClose[i] < longRebuyPrice*rebuyInputs.Stop:
			out[i] = Close
		case !(out[i-1] >= LongBuy) && shortSellLevel != nil && signal[i] < shortSellLevel[i]:
			out[i] = ShortSell
		case out[i-1] <= ShortSell && shortBuyLevel != nil && signal[i] > shortBuyLevel[i]:
			out[i] = Close
		case longRebuyEnabled && longRebuys < rebuyInputs.AllowedLongRebuys && !(out[i-1] >= LongBuy || out[i-1] == ShortSell) &&
			consecutiveUpDays >= rebuyInputs.ConsecutiveUpDays:
			out[i] = LongRebuy
			longRebuys++
			if i < seriesLen-1 {
				longRebuyPrice = rebuyInputs.DlIssue.DatasetAsColumns.AdjOpen[i+1]
			}
		default:
			if i > 0 {
				out[i] = out[i-1]
			}
		}

		if (i > 0 && out[i] == Close) && (out[i-1] >= LongBuy || out[i-1] <= ShortSell) {
			gap = TradeGap
		}
	}
	return out, nil
}

// TradeAddStop modifies the input trade signal to sell on stop.
// stopLoss is the percentage of the high price since the trade was opened that will trigger a stop.
// stopLossDelay keeps a long trade closed for this many points after the stop is triggered.
// This is to prevent a trade from being closed and then immediately re-opened.
func TradeAddStop(trade []int, stopLoss float64, stopLossDelay int, dlIssue downloader.Issue) (tradeOut []int) {
	tradeOut = make([]int, len(trade))

	bestCloseSinceBuy := 0.0
	stopTriggered := false
	stopTriggeredIndex := 0
	for i := 1; i < len(trade); i++ {
		if stopTriggered {
			tradeOut[i] = Close
			if i >= stopTriggeredIndex+stopLossDelay {
				stopTriggered = false
			}
			continue
		}

		switch {
		case trade[i-1] >= LongBuy && trade[i] >= LongBuy:
			if dlIssue.DatasetAsColumns.AdjClose[i] > bestCloseSinceBuy {
				bestCloseSinceBuy = dlIssue.DatasetAsColumns.AdjClose[i]
			}
		case trade[i-1] <= ShortSell && trade[i] <= ShortSell:
			if dlIssue.DatasetAsColumns.AdjClose[i] < bestCloseSinceBuy {
				bestCloseSinceBuy = dlIssue.DatasetAsColumns.AdjClose[i]
			}
		case trade[i-1] == Close && (trade[i] >= LongBuy || trade[i] <= ShortSell):
			if i == len(trade)-1 {
				bestCloseSinceBuy = dlIssue.DatasetAsColumns.AdjClose[i]
			} else {
				// When opening a new trade, set the stop based on the open price, since that is
				// where it was bought.
				bestCloseSinceBuy = dlIssue.DatasetAsColumns.AdjOpen[i+1]
			}
		}

		tradeOut[i] = trade[i]
		changeLong := dlIssue.DatasetAsColumns.AdjClose[i] / bestCloseSinceBuy
		changeShort := 1 / changeLong
		if trade[i] >= LongBuy && changeLong < stopLoss ||
			trade[i] <= ShortSell && changeShort < stopLoss {
			stopTriggered = true
			stopTriggeredIndex = i
			tradeOut[i] = Close
		}
	}

	return tradeOut
}
