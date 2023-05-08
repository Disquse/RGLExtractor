# RGL Extractor
[![Release](https://github.com/Disquse/RGLExtractor/actions/workflows/release.yml/badge.svg)](https://github.com/Disquse/RGLExtractor/actions/workflows/release.yml)

A tool for extracting content from Rockstar Game Launcher's (in short RGL) RAGE Packfiles (mainly just `Launcher.rpf` at this moment).
RGL uses `RPF7` (like GTAV) but encrypted with a different AES key. The key is obviously not included, but the tool can automatically find it in `Launcher.exe`.
This tool was created only for educational and data mining purposes.

## How to use
- Download the latest available [artifact](https://github.com/Disquse/RGLExtractor/actions/workflows/release.yml).
- Unzip `RGLExtractor.exe`
- Run the tool and pass required arguments. For example:
`.\RGLExtractor.exe --rgl "C:\Program Files\Rockstar Games\Launcher" --out "C:\Launcher_rpf"`

## Building
Download and install [Go](https://go.dev) (1.18+) toolchain.

```powershell
git clone "https://github.com/Disquse/RGLExtractor"
cd RGLExtractor
go build
.\RGLExtractor.exe --rgl "C:\Program Files\Rockstar Games\Launcher" --out "C:\Launcher_rpf"
```

## Thanks
- dexyfex for [CodeWalker](https://github.com/dexyfex/CodeWalker)
- 0x1F9F1 for [Swage](https://github.com/0x1F9F1/Swage)
- kelindar for [iostream](https://github.com/kelindar/iostream)

## License
MIT.
