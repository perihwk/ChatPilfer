package teams

import (
	"fmt"
	"net/http"
)

type authTransport struct {
	token     string
	userAgent string
	base      http.RoundTripper
}

func (at *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header.Set("User-Agent", at.userAgent)
	cloned.Header.Set("Authorization", fmt.Sprintf("Bearer: %s", at.token))

	return at.base.RoundTrip(cloned)
}
