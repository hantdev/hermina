PROGRAM = hermina
SOURCES = $(wildcard *.go) cmd/main.go

all: $(PROGRAM)

.PHONY: all clean $(PROGRAM)

$(PROGRAM): $(SOURCES)
	go build -ldflags "-s -w" -o ./build/$@ cmd/main.go

clean:
	rm -rf $(PROGRAM)