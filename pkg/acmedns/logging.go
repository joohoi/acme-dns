package acmedns

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

func SetupLogging(config AcmeDnsConfig) (*zap.Logger, error) {
	var logger *zap.Logger
	logformat := "console"
	if config.Logconfig.Format == "json" {
		logformat = "json"
	}
	outputPath := "stdout"
	if config.Logconfig.Logtype == "file" {
		outputPath = config.Logconfig.File
	}
	errorPath := "stderr"
	if config.Logconfig.Logtype == "file" {
		errorPath = config.Logconfig.File
	}
	zapConfigJson := fmt.Sprintf(`{
   "level": "%s",
   "encoding": "%s",
   "outputPaths": ["%s"],
   "errorOutputPaths": ["%s"],
   "encoderConfig": {
	 "timeKey": "time",
     "messageKey": "msg",
     "levelKey": "level",
     "levelEncoder": "lowercase",
	 "timeEncoder": "iso8601"
   }
 }`, config.Logconfig.Level, logformat, outputPath, errorPath)
	var zapCfg zap.Config
	if err := json.Unmarshal([]byte(zapConfigJson), &zapCfg); err != nil {
		return logger, err
	}
	logger, err := zapCfg.Build()
	return logger, err
}
