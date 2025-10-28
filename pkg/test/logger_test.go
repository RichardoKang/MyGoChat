package test

import (
	"MyGoChat/pkg/log"
	"net/http"
	"testing"

	"go.uber.org/zap"
)

func simpleHttpGet(url string) {
	log.Logger.Sugar().Debug("Trying to hit GET request", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Logger.Sugar().Errorf("Error fetching URL url=%s: %v", url, err)
	} else {
		log.Logger.Sugar().Infof("Success! statusCode=%s url=%s", resp.Status, url)
		resp.Body.Close()
	}
}

func TestInitLogger(t *testing.T) {

	log.InitLogger("./log/test.log", "debug")
	defer func(Logger *zap.Logger) {
		err := Logger.Sync()
		if err != nil {

		}
	}(log.Logger)
	simpleHttpGet("http://www.example.com")

}
