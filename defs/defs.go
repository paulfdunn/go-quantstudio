package defs

const (
	AppName = "go-quantstudio"
	GUIPort = ":8080"

	// CHANGE DEFAULTS HERE AND IN HTML FILES.
	// Length of the MA in data points.
	CvOLengthDefault    = 250
	MAHLengthDefault    = 150
	DirLengthDefault    = 200
	EMA2LengthDefaultLF = 140
	EMA2LengthDefaultHF = 110
	MA2LengthDefaultLF  = 150
	MA2LengthDefaultHF  = 40
	// The moving average is split +/- this amount; I.E. 0.05 means a buy at 5% above the MA
	// and sell at 5% below the MA.
	CvOSplitDefault = 0.04
	MAHSplitDefault = 0.04
	DirSplitDefault = 0.05
	//
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
	// TQQQ ProShares UltraPro QQQ
	TradingSymbolsDefault = "dia,idev,iefa,psq,qqq,qqqm,rsp,spy,vgt,vt,ddm,qld,sso,tqqq"
	// testing only
	// TradingSymbolsDefault = "dia,ioo,qqq,spy,vt"
)
