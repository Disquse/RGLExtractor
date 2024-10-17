package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	cmdInvalid         = 0
	cmdExtractLauncher = 1
	cmdDecryptTitles   = 2
)

type cliParams struct {
	cmdType    int
	rglPath    string
	outPath    string
	titlesPath string
}

const (
	helpCommand = "`.\\RGLExtractor.exe --rgl \"C:\\Program Files\\Rockstar Games\\Launcher\" --out \"C:\\Launcher_rpf\"`" +
		"\nor\n`.\\RGLExtractor.exe --titles \"C:\\Launcher_rpf\" --out \"C:\\titles_rgl\"`"
)

func parseParams() *cliParams {
	rglPath := flag.String("rgl", "", "Path to root folder of RGL installation")
	outPath := flag.String("out", "", "Path to output folder for extraction")
	titlesPath := flag.String("titles", "", "Path to folder with title.rgl files to decrypt")

	flag.Parse()

	cmdType := cmdInvalid

	if *titlesPath != "" {
		cmdType = cmdDecryptTitles

		pathStat, err := os.Stat(*titlesPath)
		if err == os.ErrNotExist || !pathStat.IsDir() {
			fmt.Printf("Invalid titles path: \"%s\"\n", *rglPath)
			return nil
		}
	} else if *rglPath != "" {
		cmdType = cmdExtractLauncher

		pathStat, err := os.Stat(*rglPath)
		if err == os.ErrNotExist || !pathStat.IsDir() {
			fmt.Printf("Invalid launcher path: \"%s\"\n", *rglPath)
			return nil
		}
	} else {
		fmt.Printf("You need to specify more arguments. Example:\n%s\n", helpCommand)
		return nil
	}

	if *outPath == "" {
		fmt.Printf("You need to specify output path. Example:\n%s\n", helpCommand)
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
		cmdType:    cmdType,
		rglPath:    *rglPath,
		outPath:    *outPath,
		titlesPath: *titlesPath,
	}
}

func extractLauncher(params *cliParams) error {
	rgl, err := LoadLauncher(params.rglPath)

	if err != nil {
		return err
	}

	logFunc := func(log string) {
		fmt.Println(log)
	}

	for _, packFile := range rgl.Files {
		err = packFile.extractPackFile(params.outPath, logFunc)

		if err != nil {
			return err
		}
	}

	fmt.Printf("Done! Extracted into %s\n", params.outPath)
	return nil
}

func decryptTitles(params *cliParams) error {
	decryptFile := func(filePath string) error {
		title, err := ReadTitleFromFile(filePath)
		if err != nil {
			return err
		}

		fileName := title.Name
		if fileName == "" {
			_, fileName = filepath.Split(filePath)
		}

		outPath := filepath.Join(params.outPath, fileName+".rgl.json")
		directory := filepath.Dir(outPath)

		if _, err := os.Stat(directory); os.IsNotExist(err) {
			err := os.MkdirAll(directory, 0755)

			if err != nil {
				return err
			}
		}

		file, err := os.OpenFile(outPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}

		defer file.Close()

		content := title.decrypt()
		if _, err = file.Write([]byte(content)); err != nil {
			return err
		}

		return nil
	}

	err := filepath.Walk(params.titlesPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && filepath.Ext(info.Name()) == ".rgl" {
			decryptFile(path)

		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	fmt.Printf("Done! Decrypted into %s\n", params.outPath)
	return nil
}
