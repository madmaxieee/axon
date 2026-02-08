package cache

import (
	"encoding/json"
	"os"

	"github.com/adrg/xdg"
	"github.com/madmaxieee/axon/internal/config"
	"github.com/madmaxieee/axon/internal/proto"
)

type RunData struct {
	Pattern *config.Pattern
	Flags   proto.Flags
	Input   string
	Prompt  string
}

func getLastRunDataPath() (string, error) {
	return xdg.CacheFile("axon/last_run_data.json")
}

func getLastOutputPath() (string, error) {
	return xdg.CacheFile("axon/last_output.txt")
}

func GetLastRunData() (*RunData, error) {
	path, err := getLastRunDataPath()
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var runData RunData
	err = json.Unmarshal(content, &runData)
	if err != nil {
		return nil, err
	}
	return &runData, nil
}

func SaveRunData(runData *RunData) error {
	if runData == nil || runData.Pattern == nil {
		return nil
	}
	path, err := getLastRunDataPath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(runData)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func GetLastOutput() (string, error) {
	path, err := getLastOutputPath()
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func SaveOutput(output string) error {
	path, err := getLastOutputPath()
	if err != nil {
		return err
	}
	os.WriteFile(path, []byte(output), 0644)
	return nil
}
