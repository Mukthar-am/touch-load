package configs

import(
	"github.com/creamdog/gonfig"
	"os"
	"fmt"
)

/**
	Ref link: https://github.com/creamdog/gonfig
 */

type ExampleMessageStruct struct {
	Message string
	Subject string
}

func GetFileHandler(fileAbsPath string) gonfig.Gonfig {
	gonfigFile, fileReadError := os.Open(fileAbsPath)

	// Check for file read errors
	if (fileReadError != nil) {
		fmt.Println("# Read Error:= ", fileReadError)
		os.Exit(1)
	}

	config, parserError := gonfig.FromJson(gonfigFile)	// load json
	// Check for JSON parse errors
	if parserError != nil || config == nil {
		fmt.Println("# Parse Error:= ", parserError)
	}
	defer gonfigFile.Close();

	return config
}