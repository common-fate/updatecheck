package updatecheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/common-fate/clio"
)

// waitgroup is used to ensure that Check() has finished
var waitgroup sync.WaitGroup

var checks struct {
	mu   sync.Mutex
	msgs []string
}

type checkRequest struct {
	// Application is the app we are checking for updates to.
	Application App `json:"application"`
	// Version is the current version.
	Version string `json:"version"`
	// Architecture is the operating system's architecture.
	Architecture string `json:"arch"`
	// OS is the operating system.
	OS string `json:"os"`
}

type checkResponse struct {
	// UpdateRequired is true if there is a new version available
	UpdateRequired bool `json:"updateRequired"`
	// Message to display to the user. Can include security notifications.
	Message string `json:"message"`
}

// Check for updates to the CLI application.
// Update checking happens in the background, call Print()
// to print the update message.
//
// 'prod' should be true if the build is a production build.
func Check(app App, currentVersion string, prod bool, opts ...func(*Options)) {
	o := Options{
		Client: http.DefaultClient,
		URL:    "https://update-dev.commonfate.io/check",
	}

	if prod {
		o.URL = "https://update.commonfate.io/check"
	}

	for _, opt := range opts {
		opt(&o)
	}

	if os.Getenv("GRANTED_DISABLE_UPDATE_CHECK") == "true" {
		clio.Debug("GRANTED_DISABLE_UPDATE_CHECK env var is true, skipping update check")
		return
	}

	vc, ok := loadVersionConfig(app)
	if ok && time.Now().Weekday() == vc.LastCheckForUpdates {
		clio.Debug("skipping update check until tomorrow, versionconfig=%s", vc.Path())
		return
	}

	// reset any existing messages
	checks.mu.Lock()
	defer checks.mu.Unlock()
	checks.msgs = nil

	waitgroup.Add(1)
	go doCheck(app, currentVersion, prod, vc, o)
}

// Print whether any updates are required.
func Print() {
	waitgroup.Wait()
	for _, msg := range checks.msgs {
		if msg != "" {
			clio.Info(msg)
		}
	}
}

func doCheck(app App, currentVersion string, prod bool, vc versionConfig, o Options) {
	defer waitgroup.Done()
	clio.Debug("checking for update, url=%s versionconfig=%s", o.URL, vc.Path())
	r, err := callCheckAPI(app, currentVersion, prod, o)
	if err != nil {
		clio.Debug("error when checking for updates: %s", err.Error())
		return
	}
	vc.LastCheckForUpdates = time.Now().Weekday()
	err = vc.Save()
	if err != nil {
		clio.Debug("error saving version config: %s", err.Error())
		// don't return here, keep going so that we can print a message anyway.
	}
	clio.Debugf("update required: %v, message: %v", r.UpdateRequired, r.Message)

	checks.mu.Lock()
	defer checks.mu.Unlock()
	checks.msgs = append(checks.msgs, r.Message)
}

func callCheckAPI(app App, currentVersion string, prod bool, o Options) (*checkResponse, error) {
	cr := checkRequest{
		Application:  app,
		Version:      currentVersion,
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}

	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(cr)
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("POST", o.URL, b)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", userAgent())

	res, err := o.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got invalid response from update checker API: %d", res.StatusCode)
	}

	var resp checkResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// userAgent returns a header to use in User-Agent
func userAgent() string {
	return fmt.Sprintf("cf-updatecheck-go/%s %s (%s)", getLibraryVersion(), retrieveCallInfo(), runtime.GOOS)
}

// retrieveCallInfo finds the Go package that the update was called from to include in the user agent header.
func retrieveCallInfo() string {
	pc, _, _, ok := runtime.Caller(3)
	if !ok {
		return ""
	}
	parts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	pl := len(parts)
	packageName := ""

	if parts[pl-2][0] == '(' {
		packageName = strings.Join(parts[0:pl-2], ".")
	} else {
		packageName = strings.Join(parts[0:pl-1], ".")
	}

	return packageName
}
func getLibraryVersion() (libver string) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	for _, dep := range bi.Deps {
		if dep.Path == "github.com/common-fate/updatecheck" {
			return dep.Version

		}
	}
	return ""
}
