# Use this script to run both go-quantstudio and automator, or 
# open a terminal and run:
# go build && ./go-quantstudio --logfile="" 
# open a second terminal and run:
# go run automator.go --logfile=""
go build && ./go-quantstudio &
sleep 10
cd automator
go build && ./automator