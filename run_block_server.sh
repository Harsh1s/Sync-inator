# Start blockstore servers
go run cmd/SyncinatorServerExec/main.go -s block -p 8080 -l &
go run cmd/SyncinatorServerExec/main.go -s block -p 8081 -l &
wait
