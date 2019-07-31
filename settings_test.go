package godns

import (
	"testing"
)

func TestLoadSetting(t *testing.T) {
	var settings Settings
	err := LoadSettings("./config_sample.json", &settings)

	if err != nil {
		t.Error(err.Error())
	}

	if len(settings.IPUrls) > 0 {
		t.Error("cannot load ip_url from config file")
	}

	err = LoadSettings("./file/does/not/exists", &settings)
	if err == nil {
		t.Error("file doesn't exist, should return error")
	}
}
