package main

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

var (
	errFileType    = errors.New("rpf: unsupported file type, expected RPF7")
	errEncryption  = errors.New("rpf: unsupported encryption type, expected 0xFFFFFF7")
	errNoReader    = errors.New("rpf: reader is not initialized")
	errNotReadable = errors.New("rpf: file is not ready for reading")
	errCantExtract = errors.New("rpf: can not extract entry of this type")
)

type fiPackHeader struct {
	Magic         uint32
	EntryCount    uint32
	NamesLength   uint32
	NameShift     uint8
	PlatformBit   byte
	DecryptionTag uint32
}

type fiPackEntry struct {
	NameOffset uint16
	OnDiskSize uint32
	Offset     uint32
	IsResource bool

	// Size (Binary)
	// VirtualFlags (Resource)
	// EntryIndex (Directory)
	second uint32

	// DecryptionTag (Binary)
	// PhysicalFlags (Resource)
	// EntryCount (Directory)
	third uint32
}

type fiPackFile struct {
	Path    string
	Reader  *Reader
	Header  *fiPackHeader
	Entries []*fiPackEntry
	Names   []byte
	Crypto  *aesCrypto
}

func (fi *fiPackEntry) isDirectory() bool {
	return fi.Offset == 0xFFFFFE00
}

func (fi *fiPackEntry) isResource() bool {
	return fi.IsResource
}

func (fi *fiPackEntry) isBinary() bool {
	return !fi.isResource() && !fi.isDirectory()
}

func (fi *fiPackEntry) getNameOffset() uint16 {
	return fi.NameOffset
}

func (fi *fiPackFile) isReadable() bool {
	if fi.Reader == nil || fi.Header == nil {
		return false
	}

	return true
}

func (rgl *rglInst) readPackFile(reader *Reader) (*fiPackFile, error) {
	packFile := &fiPackFile{
		Reader: reader,
		Crypto: rgl.Crypto,
	}

	header, err := packFile.readPackHeader()
	if err != nil {
		return nil, err
	}

	packFile.Header = header

	entries, err := packFile.readPackEntries()
	if err != nil {
		return nil, err
	}

	packFile.Entries = entries

	strings, err := packFile.readPackStrings()
	if err != nil {
		return nil, err
	}

	packFile.Names = strings

	return packFile, nil
}

func (fi *fiPackFile) readPackHeader() (*fiPackHeader, error) {
	if fi.Reader == nil {
		return nil, errNoReader
	}

	magic, err := fi.Reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	if magic != 0x52504637 { // should be 'RPF7'
		return nil, errFileType
	}

	entryCount, err := fi.Reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	temp, err := fi.Reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	decryptionTag, err := fi.Reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	if decryptionTag != 0xFFFFFF7 {
		return nil, errEncryption
	}

	packHeader := fiPackHeader{
		Magic:         magic,
		EntryCount:    entryCount,
		NamesLength:   (temp & 0xFFFFFFF),        // 28 bits
		NameShift:     uint8((temp >> 28) & 0x7), // 3 bits
		PlatformBit:   byte(temp & 0x80000000),   // 1 bit
		DecryptionTag: decryptionTag,
	}

	return &packHeader, nil
}

func (fi *fiPackFile) readPackEntry(reader *Reader) (*fiPackEntry, error) {
	if !fi.isReadable() {
		return nil, errNotReadable
	}

	first, err := reader.ReadUint64()
	if err != nil {
		return nil, err
	}

	second, err := reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	third, err := reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	packEntry := &fiPackEntry{
		NameOffset: uint16(first & 0xFFFF),                  // 16 bits
		OnDiskSize: uint32((first >> 16) & 0xFFFFFF),        // 24 bits
		Offset:     uint32(((first >> 40) & 0x7FFFFF) << 9), // 23 bits
		IsResource: (first >> 63) == 1,                      // 1 bit
		second:     second,
		third:      third,
	}

	return packEntry, nil
}

func (fi *fiPackFile) readPackEntries() ([]*fiPackEntry, error) {
	if !fi.isReadable() {
		return nil, errNotReadable
	}

	encrypted := make([]byte, fi.Header.EntryCount*16) // 16 bytes per entry

	if _, err := fi.Reader.Read(encrypted); err != nil {
		return nil, err
	}

	decrypted := fi.Crypto.decrypt(encrypted)

	// Create temp reader
	reader := NewReader(bytes.NewBuffer(decrypted))

	entries := make([]*fiPackEntry, fi.Header.EntryCount)

	for i := 0; i < int(fi.Header.EntryCount); i++ {
		packEntry, err := fi.readPackEntry(reader)

		if err != nil {
			return nil, err
		}

		entries[i] = packEntry
	}

	return entries, nil
}

func (fi *fiPackFile) readPackStrings() ([]byte, error) {
	if !fi.isReadable() {
		return nil, errNotReadable
	}

	encrypted := make([]byte, fi.Header.NamesLength)

	if _, err := fi.Reader.Read(encrypted); err != nil {
		return nil, err
	}

	decrypted := fi.Crypto.decrypt(encrypted)

	return decrypted, nil
}

func (rgl *rglInst) loadPackFiles() error {
	// Assume RPF files are only in root directory
	files, err := ioutil.ReadDir(rgl.Path)
	if err != nil {
		return err
	}

	for _, file := range files {
		packName := file.Name()

		if !strings.HasSuffix(packName, ".rpf") {
			continue
		}

		packFile, err := rgl.loadPackFile(packName)
		if err != nil {
			return err
		}

		rgl.Files[packName] = packFile
	}

	return nil
}

func (rgl *rglInst) loadPackFile(name string) (*fiPackFile, error) {
	filePath := path.Join(rgl.Path, name)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBuffer(content)
	reader := NewReader(buffer)

	packFile, err := rgl.readPackFile(reader)
	if err != nil {
		return nil, err
	}

	packFile.Path = filePath

	return packFile, nil
}

func (fi *fiPackFile) buildEntryPathMap() map[int]string {
	// Map for root relative paths of entries
	pathsMap := make(map[int]string)

	// Append root directory
	var entryStack []*fiPackEntry
	entryStack = append(entryStack, fi.Entries[0])

	// Build root relative paths for entries
	for {
		if len(entryStack) <= 0 {
			break
		}

		entryItem := entryStack[len(entryStack)-1]
		entryStack = entryStack[:len(entryStack)-1]

		entryName := fi.getPackEntryName(entryItem)

		startIndex := entryItem.getDirectoryEntryIndex()
		endIndex := startIndex + entryItem.getDirectoryEntryCount()

		for i := startIndex; i < endIndex; i++ {
			innerPack := fi.Entries[i]
			innerName := fi.getPackEntryName(innerPack)

			if innerPack.isDirectory() {
				entryStack = append(entryStack, innerPack)
			}

			if entryName == "" {
				pathsMap[i] = innerName
			} else {
				pathsMap[i] = fmt.Sprintf("%s\\%s", entryName, innerName)
			}
		}
	}

	return pathsMap
}

func (fi *fiPackFile) extractPackFile(outPath string, logFunc func(string)) error {
	if !fi.isReadable() {
		return errNotReadable
	}

	if logFunc != nil {
		logFunc(fmt.Sprintf("Extracting pack file \"%s\"", path.Base(fi.Path)))
	}

	entryPaths := fi.buildEntryPathMap()

	for i, entryPath := range entryPaths {
		packEntry := fi.Entries[i]

		if packEntry != nil && packEntry.isBinary() {
			if logFunc != nil {
				logFunc(fmt.Sprintf("Extracting pack entry \"%s\"", entryPath))
			}

			extractPath := path.Clean(outPath + "\\" + entryPath)
			err := fi.extractPackEntry(fi.Entries[i], extractPath)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (fi *fiPackFile) extractPackEntry(packEntry *fiPackEntry, outPath string) error {
	if !fi.isReadable() {
		return errNotReadable
	}

	// Resources are not used in RGL, so skipping for now...
	if packEntry.isDirectory() || packEntry.isResource() {
		return errCantExtract
	}

	readerOffset := fi.Reader.GetOffset()

	entryOffset := packEntry.Offset
	fi.Reader.SetOffset(int64(entryOffset))

	entrySize := int(packEntry.OnDiskSize)
	binarySize := packEntry.getBinarySize()

	if entrySize == 0 {
		entrySize = binarySize
	}

	entryContent := make([]byte, entrySize)

	if _, err := fi.Reader.Read(entryContent); err != nil {
		return err
	}

	fi.Reader.SetOffset(readerOffset)

	decryptionTag := packEntry.getBinaryDecryptionTag()

	if decryptionTag == 1 {
		entryContent = fi.Crypto.decrypt(entryContent)
	}

	// Entry is compressed
	if packEntry.OnDiskSize > 0 {
		reader := flate.NewReader(bytes.NewReader(entryContent))
		decompressed := make([]byte, binarySize)

		defer reader.Close()

		if _, err := io.ReadFull(reader, decompressed); err != nil {
			return err
		}

		entryContent = decompressed
	}

	// Some entries has no extension, let's guess using magic
	if len(strings.Split(outPath, ".")) == 1 {
		if string(entryContent[1:4]) == "PNG" || string(entryContent[6:10]) == "Exif" {
			outPath += ".png"
		} else if string(entryContent[0:3]) == "GIF" {
			outPath += ".gif"
		} else {
			outPath += ".bin"
		}
	}

	pathParts := strings.Split(outPath, "\\")
	directory := strings.Join(pathParts[:len(pathParts)-1], "\\")

	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err := os.MkdirAll(directory, 0755)

		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err = file.Write(entryContent); err != nil {
		return err
	}

	return nil
}

func (fi *fiPackFile) getPackEntryName(packEntry *fiPackEntry) string {
	startPos := uint32(packEntry.getNameOffset())
	endPos := startPos

	// Find string with null terminator, better than initializing reader
	for {
		if fi.Names[endPos] == 0x0 {
			break
		}

		if fi.Header.NamesLength <= endPos {
			return ""
		}

		endPos++
	}

	return string(fi.Names[startPos:endPos])
}

func (fi *fiPackEntry) getDirectoryEntryIndex() int {
	if !fi.isDirectory() {
		return 0
	}

	return int(fi.second)
}

func (fi *fiPackEntry) getDirectoryEntryCount() int {
	if !fi.isDirectory() {
		return 0
	}

	return int(fi.third)
}

func (fi *fiPackEntry) getBinarySize() int {
	if !fi.isBinary() {
		return 0
	}

	return int(fi.second)
}

func (fi *fiPackEntry) getBinaryDecryptionTag() int {
	if !fi.isBinary() {
		return 0
	}

	return int(fi.third)
}
