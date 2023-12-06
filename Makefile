build: 
	CGO_ENABLED=0 go build -o ./bin/koboimport cmd/kobo2gs/main.go