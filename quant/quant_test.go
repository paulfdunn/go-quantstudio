package quant

import (
	"fmt"
	"time"

	"github.com/paulfdunn/go-helper/mathh/v2"
	"github.com/paulfdunn/go-quantstudio/downloader"
)

func init() {
	Init("test")
}

func Example_abs() {
	input := []float64{0, -1.0, 1.0, -1}
	fmt.Printf("%+v", Abs(input))

	// Output:
	// [0 1 1 1]
}

func Example_annualizeGain() {
	totalGain := 1.1
	start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	ag := AnnualizedGain(totalGain, start, end)
	fmt.Printf("annualizedGain from %s to %s with total gain: %5.2f is %5.2f\n", start.Format(DateFormat), end.Format(DateFormat), totalGain, ag)

	totalGain = 1.1
	start = time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	end = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	ag = AnnualizedGain(totalGain, start, end)
	fmt.Printf("annualizedGain from %s to %s with total gain: %5.2f is %5.2f\n", start.Format(DateFormat), end.Format(DateFormat), totalGain, ag)

	totalGain = 1.1
	start = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	end = time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC)
	ag = AnnualizedGain(totalGain, start, end)
	fmt.Printf("annualizedGain from %s to %s with total gain: %5.2f is %5.2f\n", start.Format(DateFormat), end.Format(DateFormat), totalGain, ag)

	// Output:
	// annualizedGain from 2022-01-01 to 2023-01-01 with total gain:  1.10 is  1.10
	// annualizedGain from 2010-01-01 to 2023-01-01 with total gain:  1.10 is  1.01
	// annualizedGain from 2022-01-01 to 2022-07-01 with total gain:  1.10 is  1.21
}

func Example_differentiate() {
	f1 := []float64{0, 0, 0, 1, 0, 0, -1, 0, 0}
	result := Differentiate(f1)
	fmt.Printf("%+v", result)

	// Output:
	// [0 0 0 1 -1 0 -1 1 0]
}

func Example_ema_1() {
	f1 := make([]float64, 10)
	f2 := make([]float64, 40)
	f2 = OffsetSlice(1, f2)
	f3 := append(f1, f2...)
	result3, _ := EMA(10, false, f3)
	for _, v := range result3 {
		fmt.Printf("%4.3f, ", v)
	}

	// Output:
	// 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.273, 0.475, 0.625, 0.735, 0.818, 0.878, 0.924, 0.957, 0.982, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000,
}

func Example_ema_2() {
	f2 := make([]float64, 40)
	f2 = OffsetSlice(1, f2)
	result2, _ := EMA(10, false, f2)
	for _, v := range result2 {
		fmt.Printf("%4.3f, ", v)
	}

	// Output:
	// 0.273, 0.475, 0.625, 0.735, 0.818, 0.878, 0.924, 0.957, 0.982, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000,
}

func Example_ema_0() {
	f1 := make([]float64, 10)
	f2 := []float64{1.0}
	f3 := append(append(append(f1, f2...), f1...), f1...)
	result3, _ := EMA(10, false, f3)
	for _, v := range result3 {
		fmt.Printf("%4.3f, ", v)
	}

	// Output:
	// 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.273, 0.202, 0.150, 0.111, 0.082, 0.061, 0.045, 0.033, 0.025, 0.018, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000, 0.000,
}

func Example_marketClosedGain_1() {
	close := []float64{1, 1, 1, 1, 1, 1}
	open := []float64{1, 1, 1, 1, 1, 1}
	result, _ := MarketClosedGain(open, close)
	fmt.Printf("%+v", result)

	// Output:
	// [1 1 1 1 1 1]
}

func Example_marketClosedGain_2() {
	close := []float64{1, 1.1, 0.9090909090909091, 1, 1, 1}
	open := []float64{1, 1, 1, 1, 1, 1}
	result, _ := MarketClosedGain(open, close)
	fmt.Printf("%+v", result)

	// Output:
	// [1 1 0.9090909090909091 1 1 1]
}

func Example_marketOpenGain_1() {
	close := []float64{1, 1, 1, 1, 1, 1}
	open := []float64{1, 1, 1, 1, 1, 1}
	result, _ := MarketOpenGain(open, close)
	fmt.Printf("%+v", result)

	// Output:
	// [1 1 1 1 1 1]
}

func Example_marketOpenGain_2() {
	close := []float64{1, 1.1, 0.9090909090909091, 1, 1, 1}
	open := []float64{1, 1, 1, 1, 1, 1}
	result, _ := MarketOpenGain(open, close)
	fmt.Printf("%+v", result)

	// Output:
	// [1 1.1 1 1 1 1]
}

func Example_ma() {
	f1 := []float64{10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0}
	f2 := []float64{10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0}
	result, _ := MA(2, false, f1, f2)
	fmt.Printf("%+v\n", result)

	f1 = []float64{10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0}
	f2 = []float64{10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0}
	result, _ = MA(2, true, f1, f2)
	fmt.Printf("%+v\n", result)

	// Output:
	// [5 15 25 35 45 55 65 75]
	// [25 25 25 35 45 55 65 75]
}

func Example_multiplySlice() {
	f1 := []float64{1.0, 2.0, 3.0}
	result := MultiplySlice(1.1, f1)
	for i := range result {
		result[i] = mathh.Round(result[i], 4)
	}
	fmt.Printf("%+v\n", result)

	// Output:
	// [1.1 2.2 3.3]
}

func Example_multiplySliceGated() {
	result := MultiplySliceGated(
		2.0,
		[]float64{1.0, 2.0, 3.0, 4.0, 5.0},
		[]int{LongBuy, Close, Close, LongBuy, LongBuy}, Close)
	fmt.Printf("%+v\n", result)

	// Output:
	// [1 4 6 4 5]
}

func Example_multiplySlices() {
	f1 := []float64{1.0, 2.0, 3.0}
	f2 := []float64{10.0, 20.0, 30.0}
	result, _ := MultiplySlices(f1, f2)
	fmt.Printf("%+v\n", result)

	f1 = []float64{1.0, 2.0, 3.0}
	f2 = []float64{10.0, 20.0, 30.0}
	f3 := []float64{100.0, 200.0, 300.0}
	result, _ = MultiplySlices(f1, f2, f3)
	fmt.Printf("%+v\n", result)

	// Output:
	// [10 40 90]
	// [1000 8000 27000]
}

func Example_offsetSlice() {
	result := OffsetSlice(2.0, []float64{1.0, 2.0, 3.0, 4.0, 5.0})
	fmt.Printf("%+v\n", result)

	// Output:
	// [3 4 5 6 7]
}

func Example_reciprocolSlice() {
	result := ReciprocolSlice([]float64{1.0, 2.0, 4.0, 5.0})
	fmt.Printf("%+v\n", result)

	// Output:
	// [1 0.5 0.25 0.2]
}

func Example_slicesAreEqualLength() {
	f1 := []float64{1.0, 2.0, 3.0, 4.0}
	f2 := []float64{10.0, 20.0, 30.0, 40.0}
	result := SlicesAreEqualLength(f1, f2)
	if result != nil {
		fmt.Printf("result was not nil but was supposed to be")
	}

	f1 = []float64{1.0, 2.0, 3.0}
	f2 = []float64{10.0, 20.0, 30.0, 40.0}
	result = SlicesAreEqualLength(f1, f2)
	if result != nil {
		fmt.Printf("result was not nil but was supposed to be")
	}

	// Output:
	// result was not nil but was supposed to be
}
func Example_sumSlices() {
	f1 := []float64{1.0, 2.0, 3.0, 4.0}
	f2 := []float64{10.0, 20.0, 30.0}
	result, _ := SumSlices(f1, f2)
	if result != nil {
		fmt.Printf("result was not nil but was supposed to be")
	}

	f1 = []float64{1.0, 2.0, 3.0}
	f2 = []float64{10.0, 20.0, 30.0}
	result, _ = SumSlices(f1, f2)
	fmt.Printf("%+v\n", result)

	f1 = []float64{1.0, 2.0, 3.0}
	f2 = []float64{10.0, 20.0, 30.0}
	f3 := []float64{100.0, 200.0, 300.0}
	result, _ = SumSlices(f1, f2, f3)
	fmt.Printf("%+v\n", result)

	// Output:
	// [11 22 33]
	// [111 222 333]
}

func Example_tradeGain() {
	// make columns line up by using lby instead of LongBuy, cls instead of Close, and trade____ instead of trade.
	lby := LongBuy
	cls := Close
	trade____ := []int{cls, cls, cls, lby, lby, lby, cls, cls}
	close := []float64{1.0, 1.0, 1.0, 1.0, 1.0, 2.0, 2.0, 2.0}
	openn := []float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 2.0, 2.0}
	issue := downloader.Issue{}
	issue.DatasetAsColumns.AdjClose = close
	issue.DatasetAsColumns.AdjOpen = openn
	issue.DatasetAsColumns.Date = []time.Time{
		time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 3, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 4, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 6, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 7, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 8, 0, 0, 0, 0, time.UTC)}
	issue.Symbol = "test"
	_, gain, tradeG := TradeGain(2, trade____, issue)
	fmt.Printf("%5.2f %+v\n", gain, tradeG)

	trade____ = []int{cls, cls, cls, cls, lby, lby, lby, cls}
	_, gain, tradeG = TradeGain(2, trade____, issue)
	fmt.Printf("%5.2f %+v\n", gain, tradeG)

	trade____ = []int{cls, cls, cls, cls, cls, lby, lby, lby}
	_, gain, tradeG = TradeGain(2, trade____, issue)
	fmt.Printf("%5.2f %+v\n", gain, tradeG)

	// Output:
	// 2.00 [1 1 1 1 1 2 2 2]
	//  2.00 [1 1 1 1 1 2 2 2]
	//  1.00 [1 1 1 1 1 1 1 1]
}

func Example_tradeOnPrice() {
	delay := 2
	longBuyLevel := []float64{1.1, 1.1, 1.1, 1.1, 1.1, 1.1, 1.1, 1.1}
	longSellLevel := []float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0}
	shortSellLevel := []float64{0.8, 0.8, 0.8, 0.8, 0.8, 0.8, 0.8, 0.8}
	shortBuyLevel := []float64{0.9, 0.9, 0.9, 0.9, 0.9, 0.9, 0.9, 0.9}

	// Test the delay by triggering a buy prior to the delay expiring.
	price := []float64{1.0, 1.2, 1.2, 1.2, 1.2, 1.2, 1.2, 1.2}
	result, _ := TradeOnPrice(delay, price, longBuyLevel, longSellLevel, shortSellLevel, shortBuyLevel)
	fmt.Printf("%+v\n", result)

	// Test a long buy/sell.
	price = []float64{1.0, 1.0, 1.2, 1.2, 0.9, 0.9, 0.9, 0.9}
	result, _ = TradeOnPrice(delay, price, longBuyLevel, longSellLevel, shortSellLevel, shortBuyLevel)
	fmt.Printf("%+v\n", result)

	// Test a TradeGap after a long buy/sell.
	delay = 0
	price = []float64{1.2, 1.2, 0.9, 1.2, 1.2, 1.2, 1.2, 1.2}
	result, _ = TradeOnPrice(delay, price, longBuyLevel, longSellLevel, shortSellLevel, shortBuyLevel)
	fmt.Printf("%+v\n", result)

	// Test a short buy/sell.
	price = []float64{1.0, 0.7, 0.7, 1.0, 1.0, 1.0, 1.0, 1.0}
	result, _ = TradeOnPrice(delay, price, longBuyLevel, longSellLevel, shortSellLevel, shortBuyLevel)
	fmt.Printf("%+v\n", result)

	// Output:
	// [0 0 1 1 1 1 1 1]
	// [0 0 1 1 0 0 0 0]
	// [1 1 0 0 0 1 1 1]
	// [0 -1 -1 0 0 0 0 0]
}
