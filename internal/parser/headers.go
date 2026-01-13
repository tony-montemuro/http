package parser

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/tony-montemuro/http/internal/lws"
)

type parsedPragmaDirectives struct {
	Flags   []string
	Options map[string]string
}

type parsedAuthorizationCredentials struct {
	Scheme     string
	Parameters map[string]string
}

type parsedProductToken struct {
	Product string
	Version string
}

type parsedUserAgent struct {
	Products []parsedProductToken
	Comments []string
}

type parsedContentType struct {
	Type       string
	Subtype    string
	Parameters map[string]string
}

type ParsedRequestHeaders struct {
	Date            time.Time
	Pragma          parsedPragmaDirectives
	Authorization   parsedAuthorizationCredentials
	From            mail.Address
	IfModifiedSince time.Time
	Referer         string
	UserAgent       parsedUserAgent
	Allow           []string
	ContentEncoding string
	ContentLength   uint64
	ContentType     parsedContentType
	Expires         time.Time
	LastModified    time.Time
	Unrecognized    map[string]string
	Raw             map[string]string
}

type requestHeadersParser []byte

func (rh requestHeadersParser) Parse() (ParsedRequestHeaders, error) {
	headers := ParsedRequestHeaders{}
	parts := headerSplitter(rh).split()

	for _, header := range parts {
		parts := bytes.SplitN(header, []byte(":"), 2)
		if len(parts) < 2 {
			return headers, ClientError{message: fmt.Sprintf("Invalid header: cannot determine header name (%s)", header)}
		}

		name := lws.TrimRight(string(parts[0]))
		err := headerNameValidator(name).validate()
		if err != nil {
			return headers, ClientError{message: fmt.Sprintf("Invalid header: %s", err.Error())}
		}

		value := lws.TrimLeft(string(parts[1]))
		err = headerValueValidator(value).validate()
		if err != nil {
			return headers, fmt.Errorf("Invalid header: (%s)", err.Error())
		}

		err = headers.setHeader(name, value)
		if err != nil {
			return headers, ClientError{message: err.Error()}
		}
	}

	return headers, nil
}

type headerSplitter []byte

func (d headerSplitter) split() [][]byte {
	parts := [][]byte{}
	start := 0
	nextCrlf := bytes.Index(d, []byte(crlf))
	end := nextCrlf

	for nextCrlf != -1 {
		isLws, _ := lws.Check(string(d), end)
		if !isLws {
			parts = append(parts, d[start:end])
			start = end + len(crlf)
			nextCrlf = bytes.Index(d[start:], []byte(crlf))
			end = start
		} else {
			nextCrlf = bytes.Index(d[end+len(crlf):], []byte(crlf))
			end += len(crlf)
		}

		end += nextCrlf
	}

	last := d[start:]
	if len(last) > 0 {
		parts = append(parts, d[start:])
	}
	return parts
}

type headerNameValidator string

func (hn headerNameValidator) validate() error {
	return token(hn).validate()
}

type headerValueValidator string

func (hv headerValueValidator) validate() error {
	i := 0

	for i < len(hv) {
		isLws, next := lws.Check(string(hv), i)

		if isLws {
			i = next
			continue
		}

		if httpByte(hv[i]).isControl() {
			return fmt.Errorf("header value contains invalid control characters (%s)", hv)
		}

		i++
	}

	return nil
}

func (rh *ParsedRequestHeaders) setHeader(name, value string) error {
	var err error

	switch name {
	case "Date":
		err = rh.setDate(value)
	case "Pragma":
		err = rh.setPragma(value)
	case "Authorization":
		err = rh.setAuthorization(value)
	case "Referer":
		err = rh.setReferer(value)
	case "From":
		err = rh.setFrom(value)
	case "If-Modified-Since":
		err = rh.setIfModifiedSince(value)
	case "User-Agent":
		err = rh.setUserAgent(value)
	case "Allow":
		err = rh.setAllow(value)
	case "Content-Encoding":
		err = rh.setContentEncoding(value)
	case "Content-Length":
		err = rh.setContentLength(value)
	case "Expires":
		err = rh.setExpires(value)
	case "Last-Modified":
		err = rh.setLastModified(value)
	case "Content-Type":
		err = rh.setContentType(value)
	default:
		err = rh.setUnrecognized(name, value)
	}

	if err != nil {
		return err
	}

	if rh.Raw == nil {
		rh.Raw = make(map[string]string)
	}
	rh.Raw[name] = value
	return nil
}

func (rh *ParsedRequestHeaders) setDate(data string) error {
	date, err := dateParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid date header: %s", err.Error())
	}

	rh.Date = date
	return nil
}

func (rh *ParsedRequestHeaders) setPragma(data string) error {
	pragma, err := pragmaHeaderParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid pragma header: %s", err.Error())
	}

	rh.Pragma = pragma
	return nil
}

type pragmaHeaderParser string

func (p pragmaHeaderParser) parse() (parsedPragmaDirectives, error) {
	directives := parsedPragmaDirectives{Options: make(map[string]string)}
	parts := rulesExtractor(p).extract()
	if len(parts) == 0 {
		return directives, fmt.Errorf("at least one pragma directive is required (%s)", p)
	}

	for _, part := range parts {
		values := strings.SplitN(part, "=", 2)
		err := token(values[0]).validate()
		if err != nil {
			return directives, fmt.Errorf("pragma directive must be prepended with token: %s", part)
		}

		if len(values) == 2 {
			key := values[0]
			value := values[1]

			if key == "no-cache" {
				return directives, fmt.Errorf("pragma directive 'no-cache' value cannot have a value (%s)", part)
			}

			w, err := word(value).parse()
			if err != nil {
				return directives, fmt.Errorf("pragma directive value must be a word: %s", part)
			}

			directives.Options[key] = w
		} else {
			directives.Flags = append(directives.Flags, part)
		}
	}

	return directives, nil
}

func (rh *ParsedRequestHeaders) setReferer(data string) error {
	uri, err := safeUriParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid Referer header: %s", err.Error())
	}

	rh.Referer = uri
	return nil
}

func (rh *ParsedRequestHeaders) setAuthorization(data string) error {
	authorization, err := authorizationHeaderParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid Authorization header: %s", err.Error())
	}

	rh.Authorization = authorization
	return nil
}

type authorizationHeaderParser string

func (a authorizationHeaderParser) parse() (parsedAuthorizationCredentials, error) {
	credentials := parsedAuthorizationCredentials{}
	parts := authorizationHeaderSplitter(a).split()

	scheme := lws.TrimRight(parts[0])
	err := token(scheme).validate()
	if err != nil {
		return credentials, fmt.Errorf("malformed Authorization scheme (%s)", a)
	}
	credentials.Scheme = scheme

	err = credentials.setParams(parts[1])
	return credentials, err
}

type authorizationHeaderSplitter string

func (a authorizationHeaderSplitter) split() []string {
	i := 0

	for i < len(a) && !httpByte(a[i]).isTSpecial() {
		isNewLineLws, next, _ := lws.NewLine(string(a), i)
		if isNewLineLws {
			i = next
		} else {
			i++
		}
	}

	return []string{string(a[:i]), string(a[min(len(a), i+1):])}
}

func (ac *parsedAuthorizationCredentials) setParams(data string) error {
	params := make(map[string]string)

	if ac.Scheme == "Basic" {
		err := ac.setBasicSchemeParams(data)
		return err
	}

	for i, param := range rulesExtractor(data).extract() {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid auth parameter (param %d [%s])", i, data)
		}

		key := parts[0]
		err := token(key).validate()
		if err != nil {
			return fmt.Errorf("invalid auth parameter (param %d [%s])", i, data)
		}

		val, err := quotedString(parts[1]).parse()
		if err != nil {
			return fmt.Errorf("invalid auth parameter (param %d [%s])", i, data)
		}

		params[key] = val
	}

	ac.Parameters = params
	return nil
}

func (ac *parsedAuthorizationCredentials) setBasicSchemeParams(data string) error {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return fmt.Errorf("invalid credentials")
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid credentials")
	}

	userid := parts[0]
	err = token(userid).validate()
	if err != nil && len(userid) > 0 {
		return fmt.Errorf("invalid credentials")
	}

	password := parts[1]
	err = text(password).validate()
	if err != nil {
		return fmt.Errorf("invalid credentials")
	}

	params := make(map[string]string)
	params["userid"] = userid
	params["password"] = password
	ac.Parameters = params

	return nil
}

func (rh *ParsedRequestHeaders) setFrom(data string) error {
	address, err := mail.ParseAddress(data)
	if err != nil {
		return fmt.Errorf("Invalid From header: %s (%s)", err.Error(), data)
	}

	rh.From = *address
	return nil
}

func (rh *ParsedRequestHeaders) setIfModifiedSince(data string) error {
	date, err := dateParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid If-Modified-Since header: %s", err.Error())
	}

	rh.IfModifiedSince = date
	return nil
}

func (rh *ParsedRequestHeaders) setUserAgent(data string) error {
	data = lws.TrimLeft(data)
	i := 0
	userAgent := parsedUserAgent{}

	for i < len(data) {
		if data[i] == '(' {
			c, next, err := commentExtractor(data).extract(i)
			if err != nil {
				return fmt.Errorf("Invalid User-Agent header: bad comment - %s", err.Error())
			}

			err = comment(c).validate()
			if err != nil {
				return fmt.Errorf("Invalid User-Agent header: bad comment - %s", err.Error())
			}

			userAgent.Comments = append(userAgent.Comments, c)
			i = next

		} else {
			token, next := productTokenExtractor(data).extract(i)
			product, err := productTokenParser(token).parse()
			if err != nil {
				return fmt.Errorf("Invalid User-Agent header: bad product token - %s", err.Error())
			}

			userAgent.Products = append(userAgent.Products, product)
			i = next
		}
	}

	rh.UserAgent = userAgent
	return nil
}

type commentExtractor string

func (e commentExtractor) extract(start int) (string, int, error) {
	if e[start] != '(' {
		return "", 0, fmt.Errorf("comment must begin with open parenthesis (%s)", e)
	}

	score := 1
	i := start + 1
	for i < len(e) && score > 0 {
		if e[i] == '(' {
			score++
		}
		if e[i] == ')' {
			score--
		}
		i++
	}

	if score > 0 {
		return "", 0, fmt.Errorf("comment not properly closed (%s)", e)
	}

	comment := string(e[start:i])
	isLws, next := lws.Check(string(e), i)
	for isLws {
		i = next
		isLws, next = lws.Check(string(e), i)
	}

	return comment, i, nil
}

type productTokenExtractor string

func (e productTokenExtractor) extract(start int) (string, int) {
	i := start
	isLws, next := lws.Check(string(e), i)

	for i < len(e) && e[i] != '(' && !isLws {
		i++
		isLws, next = lws.Check(string(e), i)
	}

	productToken := string(e[start:i])
	for isLws {
		i = next
		isLws, next = lws.Check(string(e), i)
	}

	return productToken, i
}

func (rh *ParsedRequestHeaders) setAllow(data string) error {
	methods := rulesExtractor(data).extract()
	if len(methods) == 0 {
		return fmt.Errorf("Invalid Allow header: must include at least one method (%s)", data)
	}

	for _, m := range methods {
		err := token(m).validate()

		if err != nil {
			return fmt.Errorf("Invalid Allow header: includes unsupported methods (%s)", data)
		}
	}

	rh.Allow = methods
	return nil
}

func (rh *ParsedRequestHeaders) setContentEncoding(data string) error {
	err := token(data).validate()
	if err != nil {
		return fmt.Errorf("Invalid Content-Encoding header: malformed value (%s)", data)
	}

	lower := strings.ToLower(data)
	if lower == "x-gzip" || lower == "x-compress" {
		data = lower
	}

	rh.ContentEncoding = data
	return nil
}

func (rh *ParsedRequestHeaders) setContentLength(data string) error {
	n, err := strconv.ParseUint(data, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid Content-Length header: must be a valid unsigned 64-bit integer (%s)", data)
	}

	rh.ContentLength = n
	return nil
}

func (rh *ParsedRequestHeaders) setContentType(data string) error {
	contentType, err := contentTypeParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid Content-Type header: %s", err.Error())
	}

	rh.ContentType = contentType
	return nil
}

type contentTypeParser string

func (ct contentTypeParser) parse() (parsedContentType, error) {
	contentType := parsedContentType{}
	parts := strings.SplitN(string(ct), ";", 2)

	mediaType := strings.Split(lws.Trim(parts[0]), "/")
	if len(mediaType) != 2 {
		return contentType, fmt.Errorf("malformed media type header (%s)", ct)
	}

	err := token(mediaType[0]).validate()
	if err != nil {
		return contentType, fmt.Errorf("malformed media type (%s)", ct)
	}
	contentType.Type = mediaType[0]

	err = token(mediaType[1]).validate()
	if err != nil {
		return contentType, fmt.Errorf("malformed media subtype (%s)", ct)
	}
	contentType.Subtype = mediaType[1]

	if len(parts) == 2 {
		params, err := contentTypeParametersParser(parts[1]).parse()
		if err != nil {
			return contentType, err
		}
		contentType.Parameters = params
	}

	return contentType, nil
}

type contentTypeParametersParser string

func (ctp contentTypeParametersParser) parse() (map[string]string, error) {
	parameters := make(map[string]string)

	i := 0
	for i < len(ctp) {
		isLws, next := lws.Check(string(ctp), i)
		for isLws {
			i = next
			isLws, next = lws.Check(string(ctp), i)
		}

		attribute := []byte{}
		for i < len(ctp) && ctp[i] != '=' {
			attribute = append(attribute, ctp[i])
			i++
		}

		err := token(attribute).validate()
		if err != nil {
			return nil, fmt.Errorf("parameter attribute must be a token (%s)", ctp)
		}
		i++
		if i >= len(ctp) {
			return nil, fmt.Errorf("parameter has no value (%s)", ctp)
		}

		var value string
		if ctp[i] == '"' {
			v := []byte{ctp[i]}
			i++
			for i < len(ctp) && ctp[i] != '"' {
				v = append(v, ctp[i])
				i++
			}
			if i < len(ctp) {
				v = append(v, ctp[i])
				i++
			}

			value, err = quotedString(v).parse()
			if err != nil {
				return nil, err
			}

			isLws, next = lws.Check(string(ctp), i)
			for isLws {
				i = next
				isLws, next = lws.Check(string(ctp), i)
			}
		} else {
			v := []byte{}
			for i < len(ctp) && ctp[i] != ';' {
				v = append(v, ctp[i])
				i++
			}

			value = lws.TrimRight(string(v))
			err := token(value).validate()
			if err != nil {
				return nil, err
			}
		}

		parameters[string(attribute)] = value
		i++
	}

	return parameters, nil
}

func (rh *ParsedRequestHeaders) setExpires(data string) error {
	expires, err := dateParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid Expires header: %s", err.Error())
	}

	rh.Date = expires
	return nil

}

func (rh *ParsedRequestHeaders) setLastModified(data string) error {
	lastModified, err := dateParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid Last-Modified header: %s", err.Error())
	}

	rh.LastModified = lastModified
	return nil
}

func (rh *ParsedRequestHeaders) setUnrecognized(name, data string) error {
	err := text(data).validate()
	if err != nil {
		return fmt.Errorf("Invalid %s header: %s", name, err.Error())
	}

	if rh.Unrecognized == nil {
		rh.Unrecognized = make(map[string]string)
	}
	rh.Unrecognized[name] = data
	return nil
}
