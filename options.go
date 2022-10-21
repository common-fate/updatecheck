package updatecheck

import "net/http"

// Options allows aspects of the update checking to be customised.
type Options struct {
	Client *http.Client
	// URL is the update checking endpoint.
	URL string
}
