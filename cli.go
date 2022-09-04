package main

import (
	"flag"
	"fmt"
	"os"
)

type cliParams struct {
	rglPath string
	outPath string
}

const (
	helpCommand = "`.\\RGLExtractor.exe --rgl \"C:\\Program Files\\Rockstar Games\\Launcher\" --out \"C:\\Launcher_rpf\"`"
)

func parseParams() *cliParams {
	rglPath := flag.String("rgl", "", "Path to root folder of RGL installation")
	outPath := flag.String("out", "", "Path to output folder for extraction")

	flag.Parse()

	if *rglPath == "" {
		fmt.Printf("You need to specify launcher path. Example: %s\n", helpCommand)
		return nil
	}

	if *outPath == "" {
		fmt.Printf("You need to specify output path. Example: %s\n", helpCommand)
		return nil
	}

	rglPathStat, err := os.Stat(*rglPath)
	if err == os.ErrNotExist || !rglPathStat.IsDir() {
		fmt.Printf("Invalid launcher path: \"%s\"\n", *rglPath)
		return nil
	}

	outPathStat, err := os.Stat(*outPath)
	if err == os.ErrNotExist {
		err = os.MkdirAll(*outPath, 0755)
		if err != nil {
			fmt.Printf("Failed to create output path at: \"%s\"\n", *outPath)
			fmt.Println("Try a different place or launch extractor with the administrative rights")
			return nil
		}
	} else if err == nil && !outPathStat.IsDir() {
		fmt.Printf("Invalid output path: \"%s\"\n", *outPath)
		return nil
	}

	return &cliParams{
		rglPath: *rglPath,
		outPath: *outPath,
	}
}
