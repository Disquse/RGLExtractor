package main

import (
	"fmt"
)

func main() {
	params := parseParams()

	if params == nil {
		return
	}

	rgl, err := LoadLauncher(params.rglPath)

	if err != nil {
		panic(err)
	}

	logFunc := func(log string) {
		fmt.Println(log)
	}

	for _, packFile := range rgl.Files {
		err = packFile.extractPackFile(params.outPath, logFunc)

		if err != nil {
			panic(err)
		}
	}

	fmt.Printf("Done! Extracted into %s\n", params.outPath)
}
