package http

import (
	"fmt"
	"time"

	"github.com/tony-montemuro/http/internal/constructs"
)

type code int

type server struct {
	comments []string
	products []ProductVersion
}

type challenge struct {
	scheme string
	realm  string
	params map[string]string
}

type Methods struct {
	methods []Method
}

type responseHeaders struct {
	date            MessageTime
	pragma          PragmaDirectives
	location        Uri
	server          server
	wwwAuthenticate challenge
	allow           Methods
	contentEncoding ContentEncoding
	contentLength   ContentLength
	contentType     ContentType
	expires         MessageTime
	lastModified    MessageTime
	unrecognized    map[string]string
}

type responseBody []byte

type response struct {
	code    code
	headers responseHeaders
	body    responseBody
}

type ResponseWriter struct {
	response response
}

// For the following Status Codes, prefer the associated APIs:
//
// 301 Moved Permanently - Redirect(uri)
// 302 Moved Temporarily - RedirectTemporary(uri)
// 401 Unauhorrized - Unauthorized(scheme, realm)
func (rw *ResponseWriter) SetStatus(c int) error {
	if StatusText(c) == "" {
		return fmt.Errorf("not a valid status code")
	}

	rw.response.code = code(c)
	return nil
}

func (rw *ResponseWriter) Redirect(uri []byte) error {
	rw.SetStatus(StatusMovedPermanently)
	return rw.redirect(uri)
}

func (rw *ResponseWriter) RedirectTemporary(uri []byte) error {
	rw.SetStatus(StatusMovedTemporarily)
	return rw.redirect(uri)
}

func (rw *ResponseWriter) redirect(uri []byte) error {
	err := rw.SetLocation(uri)
	if err != nil {
		return fmt.Errorf("problem redirecting: %s", err.Error())
	}

	rw.SetBody(fmt.Appendf([]byte{}, "Resource moved to %s", uri))
	return nil
}

func (rw *ResponseWriter) Unauthorized(scheme, realm []byte) {
	rw.SetStatus(StatusUnauthorized)
	rw.SetChallenge(scheme, realm)
}

func (rw *ResponseWriter) SetDateHeader(d time.Time) {
	rw.response.headers.date.date = prepareTime(d)
}

func (rw *ResponseWriter) SetNoCache(b bool) {
	if b {
		rw.response.headers.pragma.Flags["no-cache"] = true
	} else {
		delete(rw.response.headers.pragma.Flags, "no-cache")
	}
}

func (rw *ResponseWriter) AddPragmaHeader(name, value []byte) error {
	sname := string(name)
	svalue := string(value)

	err := constructs.ValidateToken(sname)
	if err != nil {
		return err
	}

	_, err = constructs.ParseWord(svalue)
	if err != nil {
		return err
	}

	rw.response.headers.pragma.Options[sname] = svalue
	return nil
}

func (rw *ResponseWriter) SetLocation(u []byte) error {
	uri, err := parseAbsoluteUri(u)
	if err != nil {
		return err
	}

	rw.response.headers.location = uri
	return nil
}

func (rw *ResponseWriter) AddServerHeader(h []byte) error {
	pv, err := parseProductVersion(string(h))
	if err != nil {
		return err
	}

	rw.response.headers.server.products = append(rw.response.headers.server.products, pv)
	return nil
}

func (rw *ResponseWriter) AddServerHeaderComment(c []byte) error {
	scomment := string(c)

	err := constructs.ValidateComment(scomment)
	if err != nil {
		return err
	}

	rw.response.headers.server.comments = append(rw.response.headers.server.comments, scomment)
	return nil
}

func (rw *ResponseWriter) SetChallenge(scheme, realm []byte) error {
	sscheme := string(scheme)
	srealm := string(realm)

	err := constructs.ValidateToken(sscheme)
	if err != nil {
		return err
	}

	parsed, err := constructs.ParseUserQuotedString(srealm)
	if err != nil {
		return err
	}

	rw.response.headers.wwwAuthenticate.scheme = sscheme
	rw.response.headers.wwwAuthenticate.realm = parsed

	return nil
}

func (rw *ResponseWriter) AddChallengeParameter(name, value []byte) error {
	sname := string(name)
	svalue := string(value)

	err := constructs.ValidateToken(sname)
	if err != nil {
		return err
	}

	parsed, err := constructs.ParseUserQuotedString(svalue)
	if err != nil {
		return err
	}

	rw.response.headers.wwwAuthenticate.params[sname] = parsed
	return nil
}

func (rw *ResponseWriter) AddAllowHeader(m []byte) {
	rw.response.headers.allow.methods = append(rw.response.headers.allow.methods, Method(m))
}

func (rw *ResponseWriter) SetContentEncoding(ce []byte) error {
	encoding := ContentEncoding(ce)
	err := encoding.Validate()
	if err != nil {
		return err
	}

	rw.response.headers.contentEncoding = encoding
	return nil
}

func (rw *ResponseWriter) SetContentTypeHeader(main, sub []byte) error {
	smain := string(main)
	ssub := string(sub)

	err := constructs.ValidateToken(smain)
	if err != nil {
		return err
	}

	err = constructs.ValidateToken(ssub)
	if err != nil {
		return err
	}

	rw.response.headers.contentType.Type = smain
	rw.response.headers.contentType.Subtype = ssub
	return nil
}

func (rw *ResponseWriter) AddContentTypeHeaderParameter(name, value []byte) error {
	sname := string(name)
	svalue := string(value)

	err := constructs.ValidateToken(sname)
	if err != nil {
		return err
	}

	err = constructs.ValidateToken(svalue)
	if err == nil {
		rw.response.headers.contentType.Parameters[sname] = svalue
		return nil
	}

	parsed, err := constructs.ParseUserQuotedString(svalue)
	if err == nil {
		rw.response.headers.contentType.Parameters[sname] = parsed
		return nil
	}

	return fmt.Errorf("malformed parameter value")
}

func (rw *ResponseWriter) SetExpiresHeader(t time.Time) {
	rw.response.headers.expires.date = prepareTime(t)
}

func (rw *ResponseWriter) SetLastModifiedHeader(t time.Time) error {
	if t.After(time.Now()) {
		return fmt.Errorf("last modified cannot be a future timestamp")
	}

	rw.response.headers.lastModified.date = prepareTime(t)
	return nil
}

func (rw *ResponseWriter) SetHeader(name, value []byte) error {
	sname := string(name)
	svalue := string(value)

	switch sname {
	case "Date", "Pragma", "Location", "Server", "WWW-Authenticate", "Allow", "Content-Encoding", "Content-Length", "Content-Type", "Expires", "Last-Modified":
		return fmt.Errorf("please use API to set %s", name)
	default:
		err := validateHeaderName(sname)
		if err != nil {
			return err
		}

		err = validateHeaderValue(svalue)
		if err != nil {
			return err
		}

		rw.response.headers.unrecognized[sname] = svalue
	}

	return nil
}

func (rw *ResponseWriter) SetBody(data []byte) {
	rw.response.body = data
	rw.response.headers.contentLength = ContentLength(len(data))
}

func prepareTime(t time.Time) time.Time {
	return t.In(time.FixedZone("GMT", 0))
}
