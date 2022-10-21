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
	dir                 string
	app                 App
	LastCheckForUpdates time.Weekday `json:"lastCheckForUpdates"`
}

func (vc versionConfig) Path() string {
	return path.Join(vc.dir, string(vc.app)+"-update")
}

func (vc versionConfig) Save() error {
	if vc.dir == "" {
		return errors.New("version config dir was not specified")
	}
	if vc.app == "" {
		return errors.New("version config app was not specified")
	}

	err := os.MkdirAll(vc.dir, os.ModePerm)
	if err != nil {
		return err
	}

	data, err := json.Marshal(vc)
	if err != nil {
		return err
	}
	vcfile := path.Join(vc.dir, string(vc.app)+"-update")
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
	vc.dir = path.Join(cd, "commonfate")
	err = os.MkdirAll(vc.dir, os.ModePerm)
	if err != nil {
		clio.Debug("error creating commonfate config dir: %s", err.Error())
		return
	}

	vcfile := path.Join(vc.dir, string(app)+"-update")
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
