package fpv

import (
	"crypto/sha256"
	"embed"
	"fmt"
)

// Files are the sole canonical source bundle, embedded at build time so a worker
// never depends on an unrelated working-directory copy of the public viewer.
//
//go:embed index.html dist/fpv.css dist/fpv.js dist/fpv-assets.json
var Files embed.FS

func Asset(name string) ([]byte, bool) {
	switch name {
	case "fpv.css", "fpv.js", "fpv-assets.json":
	default:
		return nil, false
	}
	data, err := Files.ReadFile("dist/" + name)
	return data, err == nil
}

func AssetVersion() string {
	data, _ := Asset("fpv-assets.json")
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
