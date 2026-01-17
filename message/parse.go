package message

import (
	"bytes"
	"compress/gzip"
	"compress/lzw"
	"encoding/base64"
	"fmt"
	"io"
	"net/mail"
	"strconv"
	"strings"

	"github.com/tony-montemuro/http/internal/constructs"
	"github.com/tony-montemuro/http/internal/lws"
	"github.com/tony-montemuro/http/internal/rules"
)

type requestLineParser []byte

func (rl requestLineParser) parse() (RequestLine, error) {
	parts := bytes.Split(rl, []byte(" "))
	if len(parts) != 3 {
		return RequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: malformed request line (%s)", string(rl))}
	}

	m := Method(parts[0])
	err := m.Validate()
	if err != nil {
		return RequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: issue with request method (%s)", err.Error())}
	}

	uri, err := absPathUriParser(parts[1]).parse()
	if err != nil {
		return RequestLine{}, err
	}

	version, err := versionParser(parts[2]).parse()
	if err != nil {
		return RequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: issue with version (%s)", version)}
	}

	return RequestLine{Method: m, Uri: uri, Version: version}, nil
}

type versionParser string

func (v versionParser) parse() (string, error) {
	if len(v) < 8 {
		return string(v), fmt.Errorf("incomplete version (%s)", v)
	}

	data := strings.Split(string(v), string(constructs.ByteSeparator))
	if len(data) != 2 || !strings.Contains(data[1], ".") {
		return string(v), fmt.Errorf("could not determine version number (%s)", v)
	}

	if data[0] != "HTTP" {
		return string(v), fmt.Errorf("wrong protocol (%s)", data[0])
	}

	digits := strings.Split(data[1], ".")
	if len(digits) != 2 {
		return string(v), fmt.Errorf("malformed version number (%s)", data[1])
	}

	d1, err1 := strconv.Atoi(string(digits[0]))
	_, err2 := strconv.Atoi(string(digits[1]))
	if err1 != nil || err2 != nil {
		return string(v), fmt.Errorf("contains invalid characters (%s)", v)
	}
	if d1 == 0 {
		return string(v), fmt.Errorf("must be at least 1.0 (%s)", v)
	}

	return data[1], nil
}

type requestHeadersParser []byte

func (rh requestHeadersParser) Parse() (RequestHeaders, error) {
	headers := RequestHeaders{}
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
	nextCrlf := bytes.Index(d, []byte(constructs.Crlf))
	end := nextCrlf

	for nextCrlf != -1 {
		isLws, _ := lws.Check(string(d), end)
		if !isLws {
			parts = append(parts, d[start:end])
			start = end + len(constructs.Crlf)
			nextCrlf = bytes.Index(d[start:], []byte(constructs.Crlf))
			end = start
		} else {
			nextCrlf = bytes.Index(d[end+len(constructs.Crlf):], []byte(constructs.Crlf))
			end += len(constructs.Crlf)
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
	return constructs.Token(hn).Validate()
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

		if constructs.HttpByte(hv[i]).IsControl() {
			return fmt.Errorf("header value contains invalid control characters (%s)", hv)
		}

		i++
	}

	return nil
}

func (rh *RequestHeaders) setHeader(name, value string) error {
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

	if rh.raw == nil {
		rh.raw = make(map[string]string)
	}
	rh.raw[name] = value
	return nil
}

func (rh *RequestHeaders) setDate(data string) error {
	date, err := constructs.Date(data).Parse()
	if err != nil {
		return fmt.Errorf("Invalid date header: %s", err.Error())
	}

	rh.Date = MessageTime{date}
	return nil
}

func (rh *RequestHeaders) setPragma(data string) error {
	pragma, err := pragmaHeaderParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid pragma header: %s", err.Error())
	}

	rh.Pragma = pragma
	return nil
}

type pragmaHeaderParser string

func (p pragmaHeaderParser) parse() (PragmaDirectives, error) {
	directives := PragmaDirectives{Options: make(map[string]string)}
	parts := rules.Extractor(p).Extract()
	if len(parts) == 0 {
		return directives, fmt.Errorf("at least one pragma directive is required (%s)", p)
	}

	for _, part := range parts {
		values := strings.SplitN(part, "=", 2)
		err := constructs.Token(values[0]).Validate()
		if err != nil {
			return directives, fmt.Errorf("pragma directive must be prepended with token: %s", part)
		}

		if len(values) == 2 {
			key := values[0]
			value := values[1]

			if key == "no-cache" {
				return directives, fmt.Errorf("pragma directive 'no-cache' value cannot have a value (%s)", part)
			}

			w, err := constructs.Word(value).Parse()
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

func (rh *RequestHeaders) setReferer(data string) error {
	uri, err := safeUriParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid Referer header: %s", err.Error())
	}

	rh.Referer = uri
	return nil
}

func (rh *RequestHeaders) setAuthorization(data string) error {
	authorization, err := authorizationHeaderParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid Authorization header: %s", err.Error())
	}

	rh.Authorization = authorization
	return nil
}

type authorizationHeaderParser string

func (a authorizationHeaderParser) parse() (AuthorizationCredentials, error) {
	credentials := AuthorizationCredentials{}
	parts := authorizationHeaderSplitter(a).split()

	scheme := lws.TrimRight(parts[0])
	err := constructs.Token(scheme).Validate()
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

	for i < len(a) && !constructs.HttpByte(a[i]).IsTSpecial() {
		isNewLineLws, next, _ := lws.NewLine(string(a), i)
		if isNewLineLws {
			i = next
		} else {
			i++
		}
	}

	return []string{string(a[:i]), string(a[min(len(a), i+1):])}
}

func (ac *AuthorizationCredentials) setParams(data string) error {
	params := make(map[string]string)

	if ac.Scheme == "Basic" {
		err := ac.setBasicSchemeParams(data)
		return err
	}

	for i, param := range rules.Extractor(data).Extract() {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid auth parameter (param %d [%s])", i, data)
		}

		key := parts[0]
		err := constructs.Token(key).Validate()
		if err != nil {
			return fmt.Errorf("invalid auth parameter (param %d [%s])", i, data)
		}

		val, err := constructs.QuotedString(parts[1]).Parse()
		if err != nil {
			return fmt.Errorf("invalid auth parameter (param %d [%s])", i, data)
		}

		params[key] = val
	}

	ac.Parameters = params
	return nil
}

func (ac *AuthorizationCredentials) setBasicSchemeParams(data string) error {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return fmt.Errorf("invalid credentials")
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid credentials")
	}

	userid := parts[0]
	err = constructs.Token(userid).Validate()
	if err != nil && len(userid) > 0 {
		return fmt.Errorf("invalid credentials")
	}

	password := parts[1]
	err = constructs.Text(password).Validate()
	if err != nil {
		return fmt.Errorf("invalid credentials")
	}

	params := make(map[string]string)
	params["userid"] = userid
	params["password"] = password
	ac.Parameters = params

	return nil
}

func (rh *RequestHeaders) setFrom(data string) error {
	address, err := mail.ParseAddress(data)
	if err != nil {
		return fmt.Errorf("Invalid From header: %s (%s)", err.Error(), data)
	}

	rh.From = *address
	return nil
}

func (rh *RequestHeaders) setIfModifiedSince(data string) error {
	date, err := constructs.Date(data).Parse()
	if err != nil {
		return fmt.Errorf("Invalid If-Modified-Since header: %s", err.Error())
	}

	rh.IfModifiedSince = MessageTime{date}
	return nil
}

func (rh *RequestHeaders) setUserAgent(data string) error {
	data = lws.TrimLeft(data)
	i := 0
	userAgent := UserAgent{}

	for i < len(data) {
		if data[i] == '(' {
			c, next, err := commentExtractor(data).extract(i)
			if err != nil {
				return fmt.Errorf("Invalid User-Agent header: bad comment - %s", err.Error())
			}

			err = constructs.Comment(c).Validate()
			if err != nil {
				return fmt.Errorf("Invalid User-Agent header: bad comment - %s", err.Error())
			}

			userAgent.Comments = append(userAgent.Comments, c)
			i = next

		} else {
			token, next := productVersionExtractor(data).extract(i)
			product, err := productVersionParser(token).parse()
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

type productVersionExtractor string

func (e productVersionExtractor) extract(start int) (string, int) {
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

type productVersionParser string

func (p productVersionParser) parse() (ProductVersion, error) {
	product := ProductVersion{}
	parts := strings.Split(string(p), "/")
	if len(parts) > 2 {
		return product, fmt.Errorf("product token can only contain up to 1 forward slash (%s)", p)
	}

	err := constructs.Token(parts[0]).Validate()
	if err != nil {
		return product, fmt.Errorf("invalid product token (%s)", p)
	}
	product.Product = parts[0]

	if len(parts) == 2 {
		err := constructs.Token(parts[1]).Validate()
		if err != nil {
			return product, fmt.Errorf("invalid product token (%s)", p)
		}
		product.Version = parts[1]
	}

	return product, nil
}

func (rh *RequestHeaders) setAllow(data string) error {
	var methods []Method
	rules := rules.Extractor(data).Extract()
	if len(rules) == 0 {
		return fmt.Errorf("Invalid Allow header: must include at least one method (%s)", data)
	}

	for _, m := range rules {
		err := constructs.Token(m).Validate()

		if err != nil {
			return fmt.Errorf("Invalid Allow header: includes unsupported methods (%s)", data)
		}

		methods = append(methods, Method(m))
	}

	rh.Allow = methods
	return nil
}

func (rh *RequestHeaders) setContentEncoding(data string) error {
	var encoding ContentEncoding
	err := constructs.Token(data).Validate()
	if err != nil {
		return fmt.Errorf("Invalid Content-Encoding header: malformed value (%s)", data)
	}

	lower := ContentEncoding(strings.ToLower(data))
	err = lower.Validate()
	if err == nil {
		encoding = lower
	} else {
		encoding = ContentEncoding(data)
	}

	rh.ContentEncoding = encoding
	return nil
}

func (rh *RequestHeaders) setContentLength(data string) error {
	n, err := strconv.ParseUint(data, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid Content-Length header: must be a valid unsigned 64-bit integer (%s)", data)
	}

	rh.ContentLength = ContentLength(n)
	return nil
}

func (rh *RequestHeaders) setContentType(data string) error {
	contentType, err := contentTypeParser(data).parse()
	if err != nil {
		return fmt.Errorf("Invalid Content-Type header: %s", err.Error())
	}

	rh.ContentType = contentType
	return nil
}

type contentTypeParser string

func (ct contentTypeParser) parse() (ContentType, error) {
	contentType := ContentType{}
	parts := strings.SplitN(string(ct), ";", 2)

	mediaType := strings.Split(lws.Trim(parts[0]), "/")
	if len(mediaType) != 2 {
		return contentType, fmt.Errorf("malformed media type header (%s)", ct)
	}

	err := constructs.Token(mediaType[0]).Validate()
	if err != nil {
		return contentType, fmt.Errorf("malformed media type (%s)", ct)
	}
	contentType.Type = mediaType[0]

	err = constructs.Token(mediaType[1]).Validate()
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
	if len(ctp) == 0 {
		return nil, fmt.Errorf("parameter cannot be empty (%s)", ctp)
	}
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

		err := constructs.Token(attribute).Validate()
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

			value, err = constructs.QuotedString(v).Parse()
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
			err := constructs.Token(value).Validate()
			if err != nil {
				return nil, err
			}
		}

		parameters[string(attribute)] = value
		i++
	}

	return parameters, nil
}

func (rh *RequestHeaders) setExpires(data string) error {
	expires, err := constructs.Date(data).Parse()
	if err != nil {
		return fmt.Errorf("Invalid Expires header: %s", err.Error())
	}

	rh.Date = MessageTime{expires}
	return nil

}

func (rh *RequestHeaders) setLastModified(data string) error {
	lastModified, err := constructs.Date(data).Parse()
	if err != nil {
		return fmt.Errorf("Invalid Last-Modified header: %s", err.Error())
	}

	rh.LastModified = MessageTime{lastModified}
	return nil
}

func (rh *RequestHeaders) setUnrecognized(name, data string) error {
	err := constructs.Text(data).Validate()
	if err != nil {
		return fmt.Errorf("Invalid %s header: %s", name, err.Error())
	}

	if rh.Unrecognized == nil {
		rh.Unrecognized = make(map[string]string)
	}
	rh.Unrecognized[name] = data
	return nil
}

type requestBodyParser []byte

func (rb requestBodyParser) parse(rh RequestHeaders) ([]byte, error) {
	var body []byte
	length := rh.ContentLength

	if length > ContentLength(len(rb)) {
		return body, ClientError{message: "Content-Length header exceeds body length"}
	}

	for i := range length {
		body = append(body, rb[i])
	}

	return requestBodyDecoder(body).decode(rh.ContentEncoding)
}

type requestBodyDecoder []byte

func (d requestBodyDecoder) decode(encoding ContentEncoding) ([]byte, error) {
	var res []byte
	var err error
	fmt.Println()
	reader := bytes.NewReader([]byte(d))

	switch encoding {
	case ContentEncodingXGzip, ContentEncodingGZip:
		res, err = gzipDecode(reader)
	case ContentEncodingXCompress, ContentEncodingCompress:
		res, err = compressDecode(reader)
	default:
		res, err = io.ReadAll(reader)
	}

	if err != nil {
		err = ServerError{message: fmt.Sprintf("unexpected issue decoding body: %s", err.Error())}
	}

	return res, err
}

func gzipDecode(r io.Reader) ([]byte, error) {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func compressDecode(r io.Reader) ([]byte, error) {
	reader := lzw.NewReader(r, lzw.MSB, 8)
	defer reader.Close()

	data, err := io.ReadAll(reader)
	return data, err
}
