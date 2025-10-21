build:
    go build

build-release:
    go build -ldflags "-s -w" -tags release
