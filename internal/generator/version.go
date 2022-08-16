package generator

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type PackageFile struct {
	Version string `json:"version"`
}

func GenerateVersion(w io.Writer, packageRaw []byte) {
	var file PackageFile

	if err := json.Unmarshal(packageRaw, &file); err != nil {
		panic(err.Error())
	}

	writeStartFile(w, "vk_sdk", "")
	fmt.Fprint(w, fmt.Sprintf("const (\n\tVersion = %q\n)\n", buildActualVersion(file.Version)))
}

func buildActualVersion(v string) string {
	split := strings.Split(v, ".")
	return strings.Join(split[:len(split)-1], ".")
}
