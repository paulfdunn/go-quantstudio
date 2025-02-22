package defs

const (
	AppName = "go-quantstudio"
	GUIPort = ":8080"
	// Length of the MA in data points.
	MALengthDefault = 200
	// The moving average is split +/- this amount; I.E. 0.05 means a buy at 5% above the MA
	// and sell at 5% below the MA.
	MASplitDefault = 0.05
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
	// DIA SPDR Dow Jones Industrial Average ETF Trust
	// IEV iShares Europe ETF
	// IWB iShares Russell 1000 ETF
	// IWM iShares Russell 2000 ETF
	// QQQ Invesco QQQ Trust Series 1
	// RSP Invesco S&P 500 Eql Wght ETF
	// SPY SPDR S&P 500 ETF Trust
	// Double ETFs below
	// DDM ProShares Ultra Dow30
	// QLD ProShares Ultra QQQ
	// SSO ProShares Ultra S&P500
	// TQQQ ProShares UltraPro QQQ
	TradingSymbolsDefault = "dia,iev,iwb,iwm,qqq,rsp,spy,ddm,qld,sso,tqqq"
)
