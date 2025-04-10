# Start metastore servers
go run cmd/SyncinatorRaftServerExec/main.go -f config.json -i 0 &
go run cmd/SyncinatorRaftServerExec/main.go -f config.json -i 1 &
go run cmd/SyncinatorRaftServerExec/main.go -f config.json -i 2 &

wait
