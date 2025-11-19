package main

import "github.com/nonhumantrades/flowdb-go/cli"

func main() {
	cli := cli.New(nil)
	cli.Loop()
}
