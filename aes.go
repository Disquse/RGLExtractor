package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
)

var (
	aesKeyHash          = []byte{0x0E, 0x6B, 0x42, 0x74, 0x7E, 0xDF, 0x51, 0xDC, 0xE7, 0x8E, 0xD0, 0xA0, 0xA8, 0xFB, 0x22, 0xE9, 0x71, 0xC3, 0x16, 0x83}
	errNoExecutable     = errors.New("aes: launcher.exe does not exist")
	errNoEncryptionKeys = errors.New("aes: failed to find encryption keys")
	errClearCache       = errors.New("aes: clear cache")
)

const (
	cacheFileName    = "cache.bin"
	cacheFileVersion = 1
	cacheFileMagic   = "REcf"
)

type aesCrypto struct {
	Key    []byte
	Cipher cipher.Block
}

type cacheFile struct {
	Version byte
	Hash    []byte
	Key     []byte
}

func (rgl *rglInst) initCrypto() error {
	executable, err := ioutil.ReadFile(path.Join(rgl.Path, "launcher.exe"))
	if err != nil {
		return errNoExecutable
	}

	reader := NewReader(bytes.NewBuffer(executable))

	sha1 := sha1.New()
	if _, err := io.Copy(sha1, reader); err != nil {
		return err
	}

	// Reset offset after copying data into sha1 interface
	reader.SetOffset(0)

	// Get 20 bytes hash of sha1 sum
	currentHash := sha1.Sum(nil)[:20]

	cache, err := rgl.loadCache()

	if err != nil || !bytes.Equal(currentHash, cache.Hash) {
		cache, err = rgl.createCache(reader, currentHash)

		if err != nil {
			return errNoEncryptionKeys
		}

		err := cache.saveCache()
		if err != nil {
			// We can just continue without saving...
			fmt.Printf("Failed to save cache file: %s", err)
		}
	}

	cipher, err := aes.NewCipher(cache.Key)
	if err != nil {
		return err
	}

	rgl.Crypto = &aesCrypto{
		Key:    cache.Key,
		Cipher: cipher,
	}

	return nil
}

func (rgl *rglInst) loadCache() (*cacheFile, error) {
	if _, err := os.Stat(cacheFileName); err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(cacheFileName)
	if err != nil {
		return nil, err
	}

	if len(content) < (4 + 1 + 20 + 32) { // Magic, version, hash, keys
		return nil, errClearCache
	}

	reader := NewReader(bytes.NewBuffer(content))

	magic := make([]byte, 4)
	if _, err = reader.Read(magic); err != nil {
		return nil, err
	}

	if string(magic) != cacheFileMagic { // 'RGL Extractor cache file'
		return nil, errClearCache
	}

	version, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	if version == cacheFileVersion { // Current version reader
		hash := make([]byte, 20)
		if _, err := reader.Read(hash); err != nil {
			return nil, err
		}

		key := make([]byte, 32)
		if _, err := reader.Read(key); err != nil {
			return nil, err
		}

		cache := &cacheFile{
			Version: version,
			Hash:    hash,
			Key:     key,
		}

		return cache, nil
	}

	return nil, errClearCache
}

func (rgl *rglInst) createCache(reader *Reader, hash []byte) (*cacheFile, error) {
	sha1 := sha1.New()

	var offset int64
	buffer := make([]byte, 32)

	for {
		_, err := reader.Read(buffer)

		if err != nil {
			return nil, err
		}

		sha1.Reset()
		sha1.Write(buffer)

		if bytes.Equal(aesKeyHash, sha1.Sum(nil)) {
			break
		}

		offset += 8
		reader.SetOffset(offset)
	}

	cacheFile := &cacheFile{
		Version: cacheFileVersion,
		Hash:    hash,
		Key:     buffer,
	}

	return cacheFile, nil
}

func (cf *cacheFile) saveCache() error {
	file, err := os.OpenFile(cacheFileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err = file.Write([]byte(cacheFileMagic)); err != nil {
		return err
	}

	if _, err = file.Write([]byte{cf.Version}); err != nil {
		return err
	}

	if _, err = file.Write(cf.Hash); err != nil {
		return err
	}

	if _, err = file.Write(cf.Key); err != nil {
		return err
	}

	return nil
}

func (aes *aesCrypto) decrypt(data []byte) []byte {
	length := len(data) - len(data)%16
	result := make([]byte, len(data))
	blockSize := 16

	for bs, be := 0, blockSize; bs < length; bs, be = bs+blockSize, be+blockSize {
		aes.Cipher.Decrypt(result[bs:be], data[bs:be])
	}

	rest := data[length:]
	if len(rest) > 0 {
		result = append(result[:length], rest[:]...)
	}

	return result
}
