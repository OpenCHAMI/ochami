package client

import (
	"encoding/json"
	"fmt"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/format"
	"io"
	"net/http"
)

var (
	UnsuccessfulHTTPError = fmt.Errorf("unsuccessful HTTP status")
	NilMapPointerError    = fmt.Errorf("nil map pointer")
)

type HTTPHeaders map[string][]string
type HTTPBody []byte

// HTTPEnvelope represents a subset of the http.Response struct that only
// contains relevant members. It is used as the return values of client
// functions that make requests so that the caller need not import the http
// module.
type HTTPEnvelope struct {
	Status     string // e.g. "200 OK"
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.0"
	Headers    *HTTPHeaders
	Body       HTTPBody
}

// NewHTTPHeaders returns a pointer to a new HTTPHeaders.
func NewHTTPHeaders() *HTTPHeaders {
	return &HTTPHeaders{}
}

// Add adds the key, value pair to the header, appending to any existing values
// associated with the key. The value is not processed in any way before being
// added. If the recipient HTTPHeaders pointer is nil, an error is returned.
func (h *HTTPHeaders) Add(key, value string) error {
	if h == nil {
		return NilMapPointerError
	} else {
		(*h)[key] = append((*h)[key], value)
	}
	return nil
}

// SetAuthorization takes a token and adds it as an authentication header to the
// HTTPHeaders map. If the HTTPHeaders map is nil, an error is returned.
func (h *HTTPHeaders) SetAuthorization(token string) error {
	if h == nil {
		return NilMapPointerError
	}
	if err := h.Add("Authorization", fmt.Sprintf("Bearer %s", token)); err != nil {
		return fmt.Errorf("could not set authorization token in HTTPHeaders: %w", err)
	}
	return nil
}

// SetContentType takes a content type string (e.g. "text/plain") and sets the
// "Content-Type" header to it in the HTTPHeaders map.
func (h *HTTPHeaders) SetContentType(ct string) error {
	if h == nil {
		return NilMapPointerError
	}
	if err := h.Add("Content-Type", ct); err != nil {
		return fmt.Errorf("could not set Content-Type in HTTPHeaders: %w", err)
	}
	return nil
}

// NewHTTPEnvelopeFromResponse takes a pointer to an http.Response and returns a
// populated HTTPEnvelope. If res is nil or there is an error reading the
// response body, an error is returned. Importantly, this function closes the
// response body after reading it so it should not already have been closed
// before calling this function.
func NewHTTPEnvelopeFromResponse(res *http.Response) (HTTPEnvelope, error) {
	var henv HTTPEnvelope
	if res != nil {
		henv = HTTPEnvelope{
			Status:     res.Status,
			StatusCode: res.StatusCode,
			Proto:      res.Proto,
		}
		headers := &HTTPHeaders{}
		for key, vals := range res.Header {
			(*headers)[http.CanonicalHeaderKey(key)] = vals
		}
		henv.Headers = headers

		var body HTTPBody
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return henv, fmt.Errorf("could not read HTTP body: %w", err)
		}
		if err := res.Body.Close(); err != nil {
			return henv, fmt.Errorf("error closing response body: %w", err)
		}
		henv.Body = body

		return henv, nil
	} else {
		return henv, fmt.Errorf("HTTP response was nil")
	}
}

// FormatBody takes an HTTPBody and marshals it into the format specified,
// returning the resulting bytes. If an error occurs during
// marshalling/unmarshalling or the format is unsupported, an error occurs.
func FormatBody(body HTTPBody, outFormat format.DataFormat) ([]byte, error) {
	var jmap interface{}
	if err := json.Unmarshal(body, &jmap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal HTTP body: %w", err)
	}

	return format.FormatData(jmap, outFormat)
}

func (he HTTPEnvelope) CheckResponse() error {
	statusOK := he.StatusCode >= 200 && he.StatusCode < 300
	if statusOK {
		log.Logger.Info().Msgf("Response status: %s %s", he.Proto, he.Status)
		return nil
	} else {
		if len(he.Body) > 0 {
			return fmt.Errorf("%w: %s %s: %s", UnsuccessfulHTTPError, he.Proto, he.Status, string(he.Body))
		} else {
			return fmt.Errorf("%w: %s %s", UnsuccessfulHTTPError, he.Proto, he.Status)
		}
	}
}
