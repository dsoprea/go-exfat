package exfat

import (
	"os"
	"path"
)

var (
	AssetPath = ""
)

func init() {
	goPath := os.Getenv("GOPATH")
	projectPath := path.Join(goPath, "src", "github.com", "dsoprea", "go-exfat")
	AssetPath = path.Join(projectPath, "test", "assets")
}
