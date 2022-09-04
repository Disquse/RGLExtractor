package main

type rglInst struct {
	// Path to RGL installation
	Path string

	// Pack files, usually just "Launcher.rpf"
	Files map[string]*fiPackFile

	// AES crypto instance
	Crypto *aesCrypto
}

func LoadLauncher(rootPath string) (*rglInst, error) {
	rgl := rglInst{
		Path:  rootPath,
		Files: map[string]*fiPackFile{},
	}

	var err error

	err = rgl.initCrypto()
	if err != nil {
		return nil, err
	}

	err = rgl.loadPackFiles()
	if err != nil {
		return nil, err
	}

	return &rgl, nil
}
