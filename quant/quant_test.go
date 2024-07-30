package quant

import (
	"fmt"
	"time"

	"github.com/paulfdunn/go-helper/mathh"
	"github.com/paulfdunn/go-quantstudio/downloader"
)

func init() {
	Init("test")
}

func Example_AnnualizeGain() {
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

func Example_MA() {
	f1 := []float64{10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0}
	f2 := []float64{10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0}
	result := MA(2, false, f1, f2)
	fmt.Printf("%+v\n", result)

	f1 = []float64{10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0}
	f2 = []float64{10.0, 20.0, 30.0, 40.0, 50.0, 60.0, 70.0, 80.0}
	result = MA(2, true, f1, f2)
	fmt.Printf("%+v\n", result)

	// Output:
	// [5 15 25 35 45 55 65 75]
	// [25 25 25 35 45 55 65 75]
}

func Example_MultiplySlice() {
	f1 := []float64{1.0, 2.0, 3.0}
	result := MultiplySlice(1.1, f1)
	for i := range result {
		result[i] = mathh.Round(result[i], 4)
	}
	fmt.Printf("%+v\n", result)

	// Output:
	// [1.1 2.2 3.3]
}

func Example_MultiplySliceGated() {
	result := MultiplySliceGated(
		2.0,
		[]float64{1.0, 2.0, 3.0, 4.0, 5.0},
		[]int{Buy, Sell, Sell, Buy, Buy}, Sell)
	fmt.Printf("%+v\n", result)

	// Output:
	// [1 4 6 4 5]
}

func Example_MultiplySlices() {
	f1 := []float64{1.0, 2.0, 3.0}
	f2 := []float64{10.0, 20.0, 30.0}
	result := MultiplySlices(f1, f2)
	fmt.Printf("%+v\n", result)

	f1 = []float64{1.0, 2.0, 3.0}
	f2 = []float64{10.0, 20.0, 30.0}
	f3 := []float64{100.0, 200.0, 300.0}
	result = MultiplySlices(f1, f2, f3)
	fmt.Printf("%+v\n", result)

	// Output:
	// [10 40 90]
	// [1000 8000 27000]
}

func Example_OffsetSlice() {
	result := OffsetSlice(2.0, []float64{1.0, 2.0, 3.0, 4.0, 5.0})
	fmt.Printf("%+v\n", result)

	// Output:
	// [3 4 5 6 7]
}

func Example_ReciprocolSlice() {
	result := ReciprocolSlice([]float64{1.0, 2.0, 4.0, 5.0})
	fmt.Printf("%+v\n", result)

	// Output:
	// [1 0.5 0.25 0.2]
}

func Example_SlicesAreEqualLength() {
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
func Example_SumSlices() {
	f1 := []float64{1.0, 2.0, 3.0, 4.0}
	f2 := []float64{10.0, 20.0, 30.0}
	result := SumSlices(f1, f2)
	if result != nil {
		fmt.Printf("result was not nil but was supposed to be")
	}

	f1 = []float64{1.0, 2.0, 3.0}
	f2 = []float64{10.0, 20.0, 30.0}
	result = SumSlices(f1, f2)
	fmt.Printf("%+v\n", result)

	f1 = []float64{1.0, 2.0, 3.0}
	f2 = []float64{10.0, 20.0, 30.0}
	f3 := []float64{100.0, 200.0, 300.0}
	result = SumSlices(f1, f2, f3)
	fmt.Printf("%+v\n", result)

	// Output:
	// [11 22 33]
	// [111 222 333]
}

func Example_tradeGain() {
	// make columns line up by using sel instead of Sell, and trade____ instead of trade.
	sel := Sell
	trade____ := []int{sel, sel, sel, Buy, Buy, Buy, sel, sel}
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

	trade____ = []int{sel, sel, sel, sel, Buy, Buy, Buy, sel}
	_, gain, tradeG = TradeGain(2, trade____, issue)
	fmt.Printf("%5.2f %+v\n", gain, tradeG)

	trade____ = []int{sel, sel, sel, sel, sel, Buy, Buy, Buy}
	_, gain, tradeG = TradeGain(2, trade____, issue)
	fmt.Printf("%5.2f %+v\n", gain, tradeG)

	// Output:
	// 2.00 [1 1 1 1 1 2 2 2]
	//  2.00 [1 1 1 1 1 2 2 2]
	//  1.00 [1 1 1 1 1 1 1 1]
}
