package main

import (
	"ss/cmd"
	"ss/internal/progress"
)

func main() {
	progress.Init()
	cmd.Execute()
}
