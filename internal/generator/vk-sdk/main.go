package main

import (
	"flag"
	"github.com/elias506/vk-sdk/internal/generator"
	"io"
	"net/http"
	"os"
	"os/exec"
)

const (
	methodsFile   = `https://raw.githubusercontent.com/elias506/vk-api-schema/master/methods.json`
	errorsFile    = `https://raw.githubusercontent.com/elias506/vk-api-schema/master/errors.json`
	objectsFile   = `https://raw.githubusercontent.com/elias506/vk-api-schema/master/objects.json`
	responsesFile = `https://raw.githubusercontent.com/elias506/vk-api-schema/master/responses.json`
	packageFile   = `https://raw.githubusercontent.com/elias506/vk-api-schema/master/package.json`
)

func getRawFromAddr(addr string) []byte {
	resp, err := http.Get(addr)

	if err != nil {
		panic(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		panic("code is no OK: " + resp.Status)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		panic(err.Error())
	}

	return body
}

func goFmt(path string) {
	if *noformat {
		return
	}

	cmd := exec.Command("go", "fmt", path)

	err := cmd.Run()

	if err != nil {
		panic(err.Error())
	}
}

var noformat = flag.Bool("noformat", false, "do not run 'gofmt -w' on output file")

func main() {
	flag.Parse()

	genVersion("version.go")
	genErrors("error_codes.go")
	genObjects("objects.go", "objects_test.go")
	genResponses("responses.go", "responses_test.go")
	genMethods("methods.go", "methods_test.go")
}

func genErrors(file string) {
	defer goFmt(file)

	e, err := os.Create(file)

	if err != nil {
		panic(err.Error())
	}
	defer e.Close()

	generator.GenerateErrors(e, getRawFromAddr(errorsFile))
}

func genVersion(file string) {
	defer goFmt(file)

	v, err := os.Create(file)
	if err != nil {
		panic(err.Error())
	}
	defer v.Close()

	generator.GenerateVersion(v, getRawFromAddr(packageFile))
}

func genObjects(file, testFile string) {
	defer goFmt(file)
	defer goFmt(testFile)

	o, err := os.Create(file)
	if err != nil {
		panic(err.Error())
	}
	defer o.Close()

	oTest, err := os.Create(testFile)
	if err != nil {
		panic(err.Error())
	}
	defer oTest.Close()

	generator.GenerateObjects(o, oTest, getRawFromAddr(objectsFile))
}

func genResponses(file, testFile string) {
	defer goFmt(file)
	defer goFmt(testFile)

	o, err := os.Create(file)
	if err != nil {
		panic(err.Error())
	}
	defer o.Close()

	oTest, err := os.Create(testFile)
	if err != nil {
		panic(err.Error())
	}
	defer oTest.Close()

	generator.GenerateObjects(o, oTest, getRawFromAddr(responsesFile))
}

func genMethods(file, testFile string) {
	defer goFmt(file)
	defer goFmt(testFile)

	m, err := os.Create(file)
	if err != nil {
		panic(err.Error())
	}
	defer m.Close()

	mTest, err := os.Create(testFile)
	if err != nil {
		panic(err.Error())
	}
	defer m.Close()

	generator.GenerateMethods(m, mTest, getRawFromAddr(methodsFile))
}
