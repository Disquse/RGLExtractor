package main

func main() {
	params := parseParams()
	if params == nil {
		return
	}

	var err error

	switch params.cmdType {
	case cmdDecryptTitles:
		err = decryptTitles(params)
	case cmdExtractLauncher:
		err = extractLauncher(params)
	}

	if err != nil {
		panic(err)
	}
}
