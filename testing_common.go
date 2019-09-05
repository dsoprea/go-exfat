package exfat

import (
	"os"
	"path"
)

var (
	assetPath = ""
)

func init() {
	goPath := os.Getenv("GOPATH")
	projectPath := path.Join(goPath, "src", "github.com", "dsoprea", "go-exfat")
	assetPath = path.Join(projectPath, "test", "assets")
}
