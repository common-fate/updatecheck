package updatecheck

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"time"

	"github.com/common-fate/clio"
)

type versionConfig struct {
	app                 App
	LastCheckForUpdates time.Weekday `json:"lastCheckForUpdates"`
}

func (vc versionConfig) Save() error {
	cd, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	cf := path.Join(cd, "commonfate")
	err = os.MkdirAll(cf, os.ModePerm)
	if err != nil {
		return err
	}

	data, err := json.Marshal(vc)
	if err != nil {
		return err
	}
	vcfile := path.Join(cf, string(vc.app)+"-update")
	os.WriteFile(vcfile, data, 0700)
	return nil
}

func loadVersionConfig(app App) (vc versionConfig) {
	vc.app = app
	cd, err := os.UserConfigDir()
	if err != nil {
		clio.Debug("error loading user config dir: %s", err.Error())
		return
	}
	cf := path.Join(cd, "commonfate")
	err = os.MkdirAll(cf, os.ModePerm)
	if err != nil {
		clio.Debug("error creating commonfate config dir: %s", err.Error())
		return
	}

	vcfile := path.Join(cf, string(app)+"-update")
	if _, err := os.Stat(vcfile); errors.Is(err, os.ErrNotExist) {
		clio.Debug("version config file does not exist: %s", vcfile)
		return
	}

	data, err := os.ReadFile(vcfile)
	if err != nil {
		clio.Debug("error reading version config: %s", err.Error())
		return
	}
	err = json.Unmarshal(data, &vc)
	if err != nil {
		clio.Debug("error unmarshalling version config: %s", err.Error())
		return
	}
	return
}
