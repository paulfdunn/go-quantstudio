package defs

const (
	AppName = "go-quantstudio"
	GUIPort = ":8080"

	// CHANGE DEFAULTS HERE AND IN HTML FILES.
	// Length of the MA in data points.
	CvOLengthDefault = 250
	// The moving average is split +/- this amount; I.E. 0.05 means a buy at 5% above the MA
	// and sell at 5% below the MA.
	CvOSplitDefault = 0.04

	MAHLengthDefault     = 400
	MAHSplitDefault      = 0.04
	MAHShortShiftDefault = 0.8
	MAHStopLoss          = 0.8
	MAHStopLossDelay     = 15
	MAHLongQuickBuy      = true
	MAHEMA               = false
	// Update defaults for EMA in chartMA2.js:updateValues()
	MA2LengthDefaultLF   = 150
	MA2LengthDefaultHF   = 40
	MA2ShortShiftDefault = 0.9
	MA2StopLoss          = 0.8
	MA2StopLossDelay     = 15
	MA2LongQuickBuy      = true
	MA2EMA               = false

	// Symbols being added for use in analysis. These symbols will always be downloaded, but only
	// used as inputs for quantitative analysis with TradingSymbolsDefault
	//
	// ^fvx Treasury Yield 5 Years
	// ^tnx Treasury Yield 10 Years
	// AnalysisSymbols = "^fvx,^tnx"
	AnalysisSymbols = ""
	//
	// Symbols for trading
	//
	// DIA (0.16%) SPDR Dow Jones Industrial Average ETF Trust
	// IDEV (0.04%) iShares Core MSCI International Developed Markets ETF (large-, mid- and small-capitalization developed market equities, excluding the United States)
	// IEFA (0.07%) iShares Core MSCI EAFE ETF large-, mid- and small-capitalization developed market equities, excluding the U.S. and Canada. (Not currency hedged; HEFA is the equivalent ETF hedged in USD.)
	// INTF (0.16%) ISHARES INTERNATIONAL EQUITY FACTOR ETF - track the investment results of the STOXX International Equity Factor Index
	// PSQ (0.95%) ProShares Short QQQ
	// QQQ (0.2%) Invesco QQQ Trust Series 1 - lower spreads and most liquidity Nasdaq 100 ETF
	// QQQM (0.15%) Invesco Nasdaq 100 ETF - similar to QQQ, but lower cost, less liquidity, less history
	// RSP (0.2%) Invesco S&P 500 Eql Wght ETF
	// SPY (0.09%) SPDR S&P 500 ETF Trust
	// VGT (0.09%) Vanguard Information Technology Index Fund ETF Shares - QQQ like
	// VT (0.06%) Vanguard Total World Stock ETF
	// Double ETFs below
	// DDM ProShares Ultra Dow30
	// QLD ProShares Ultra QQQ
	// SSO ProShares Ultra S&P500
	// Triple ETFs below
	// TQQQ ProShares UltraPro QQQ
	TradingSymbolsDefault = "^tnx,dia,idev,iefa,intf,psq,qqq,qqqm,rsp,spy,vgt,vt,ddm,qld,sso,tqqq"
	// testing only
	// TradingSymbolsDefault = "dia,ioo,qqq,spy,vt"
)

// Bond options for when stocks aren't so great.

// Ultrashort Bond
// BIL SPDR BLOOMBERG 1-3 MONTH T-BILL ETF
// FLOT ISHARES FLOATING RATE BOND ETF
// JPST JPMORGAN ULTRA-SHORT INCOME ETF
//
// Short-Term Inflation Protected Bond
// STIP ISHARES 0-5 YEAR TIPS BOND ETF
//
// Corporate Bond
// VCIT VANGUARD INTERMEDIATE-TERM CORPORATE BOND ETF
//
// Emerging Markets Bond
// EMB ISHARES JP MORGAN USD EMERGING MARKETS BOND ETF
//
// High Yield Bond
// SHYG ISHARES 0-5 YEAR HIGH YIELD CORPORATE BOND ETF
