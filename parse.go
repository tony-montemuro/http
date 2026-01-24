package http

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/lzw"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/tony-montemuro/http/internal/constructs"
	"github.com/tony-montemuro/http/internal/lws"
	"github.com/tony-montemuro/http/internal/rules"
)

func parseRequest(conn net.Conn, server Server) (*Request, error) {
	conn.SetReadDeadline(time.Now().Add(time.Duration(server.ReadTimeout) * time.Millisecond))
	defer conn.SetReadDeadline(time.Time{})

	limitedReader := &io.LimitedReader{
		R: conn,
		N: int64(server.MaxHeaderBytes),
	}
	reader := bufio.NewReader(limitedReader)
	lineBuf, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	if !bytes.HasSuffix(lineBuf, []byte(constructs.Crlf)) {
		return nil, ClientError{message: "malformed header suffix"}
	}

	line, err := parseRequestLine(bytes.Trim(lineBuf, constructs.Crlf))
	if err != nil {
		return nil, err
	}

	var headerBuf bytes.Buffer
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if line == "\r\n" {
			break
		}

		headerBuf.WriteString(line)
	}

	headers, err := parseRequestHeaders(bytes.Trim(headerBuf.Bytes(), constructs.Crlf))
	if err != nil {
		return nil, err
	}
	if headers.ContentLength > ContentLength(server.MaxBodyBytes) {
		return nil, ClientError{message: fmt.Sprintf("Content-Length exceeds max allowed by server: %d", server.MaxBodyBytes)}
	}

	bodyBytes := make([]byte, headers.ContentLength)
	_, err = io.ReadFull(reader, bodyBytes)
	if err != nil {
		return nil, err
	}

	body, err := parseRequestBody(bodyBytes, headers)
	if err != nil {
		return nil, err
	}

	return &Request{Line: line, Headers: headers, Body: body}, nil
}

func parseRequestLine(data []byte) (RequestLine, error) {
	parts := bytes.Split(data, []byte(" "))
	if len(parts) != 3 {
		return RequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: malformed request line (%s)", data)}
	}

	m := Method(parts[0])
	err := m.Validate()
	if err != nil {
		return RequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: issue with request method (%s)", err.Error())}
	}

	uri, err := parseRelativeUri(parts[1])
	if err != nil {
		return RequestLine{}, err
	}

	if uri.getPathForm() != AbsPath {
		return RequestLine{}, fmt.Errorf("Invalid request line: issue with uri (uri must be in the form of an absolute path)")
	}

	version, err := parseVersion(string(parts[2]))
	if err != nil {
		return RequestLine{}, ClientError{message: fmt.Sprintf("Invalid request line: issue with version (%s)", version)}
	}

	return RequestLine{Method: m, Uri: uri, Version: version}, nil
}

func parseVersion(data string) (string, error) {
	if len(data) < 8 {
		return data, fmt.Errorf("incomplete version (%s)", data)
	}

	parts := strings.Split(data, string(constructs.ByteSeparator))
	if len(parts) != 2 || !strings.Contains(parts[1], ".") {
		return data, fmt.Errorf("could not determine version number (%s)", data)
	}

	if parts[0] != "HTTP" {
		return data, fmt.Errorf("wrong protocol (%s)", parts[0])
	}

	digits := strings.Split(parts[1], ".")
	if len(digits) != 2 {
		return data, fmt.Errorf("malformed version number (%s)", parts[1])
	}

	d1, err1 := strconv.Atoi(string(digits[0]))
	_, err2 := strconv.Atoi(string(digits[1]))
	if err1 != nil || err2 != nil {
		return data, fmt.Errorf("contains invalid characters (%s)", data)
	}
	if d1 == 0 {
		return data, fmt.Errorf("must be at least 1.0 (%s)", data)
	}

	return parts[1], nil
}

func parseRequestHeaders(data []byte) (RequestHeaders, error) {
	headers := RequestHeaders{}
	parts := splitRequestHeaders(data)

	for _, header := range parts {
		parts := bytes.SplitN(header, []byte(":"), 2)
		if len(parts) < 2 {
			return headers, ClientError{message: fmt.Sprintf("Invalid header: cannot determine header name (%s)", header)}
		}

		name := lws.TrimRight(string(parts[0]))
		err := validateHeaderName(name)
		if err != nil {
			return headers, ClientError{message: fmt.Sprintf("Invalid header: %s", err.Error())}
		}

		value := lws.TrimLeft(string(parts[1]))
		err = validateHeaderValue(value)
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

func splitRequestHeaders(data []byte) [][]byte {
	parts := [][]byte{}
	start := 0
	nextCrlf := bytes.Index(data, []byte(constructs.Crlf))
	end := nextCrlf

	for nextCrlf != -1 {
		isLws, _ := lws.Check(string(data), end)
		if !isLws {
			parts = append(parts, data[start:end])
			start = end + len(constructs.Crlf)
			nextCrlf = bytes.Index(data[start:], []byte(constructs.Crlf))
			end = start
		} else {
			nextCrlf = bytes.Index(data[end+len(constructs.Crlf):], []byte(constructs.Crlf))
			end += len(constructs.Crlf)
		}

		end += nextCrlf
	}

	last := data[start:]
	if len(last) > 0 {
		parts = append(parts, data[start:])
	}
	return parts
}

func validateHeaderName(data string) error {
	return constructs.ValidateToken(data)
}

func validateHeaderValue(data string) error {
	i := 0

	for i < len(data) {
		isLws, next := lws.Check(data, i)

		if isLws {
			i = next
			continue
		}

		if constructs.HttpByte(data[i]).IsControl() {
			return fmt.Errorf("header value contains invalid control characters (%s)", data)
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
	date, err := constructs.ParseDate(data)
	if err != nil {
		return fmt.Errorf("Invalid date header: %s", err.Error())
	}

	rh.Date = MessageTime{date}
	return nil
}

func (rh *RequestHeaders) setPragma(data string) error {
	pragma, err := parsePragmaDirectives(data)
	if err != nil {
		return fmt.Errorf("Invalid pragma header: %s", err.Error())
	}

	rh.Pragma = pragma
	return nil
}

func parsePragmaDirectives(data string) (PragmaDirectives, error) {
	directives := PragmaDirectives{Options: make(map[string]string), Flags: make(map[string]bool)}
	parts := rules.Extract(data)
	if len(parts) == 0 {
		return directives, fmt.Errorf("at least one pragma directive is required (%s)", data)
	}

	for _, part := range parts {
		values := strings.SplitN(part, "=", 2)
		err := constructs.ValidateToken(values[0])
		if err != nil {
			return directives, fmt.Errorf("pragma directive must be prepended with token: %s", part)
		}

		if len(values) == 2 {
			key := values[0]
			value := values[1]

			if key == "no-cache" {
				return directives, fmt.Errorf("pragma directive 'no-cache' value cannot have a value (%s)", part)
			}

			w, err := constructs.ParseWord(value)
			if err != nil {
				return directives, fmt.Errorf("pragma directive value must be a word: %s", part)
			}

			directives.Options[key] = w
		} else {
			directives.Flags[part] = true
		}
	}

	return directives, nil

}

func (rh *RequestHeaders) setReferer(data string) error {
	uri, err := parseUri([]byte(data))
	if err != nil {
		return fmt.Errorf("Invalid Referer header: %s", err.Error())
	}

	rh.Referer = uri
	return nil
}

func (rh *RequestHeaders) setAuthorization(data string) error {
	authorization, err := parseAuthorizationCredentials(data)
	if err != nil {
		return fmt.Errorf("Invalid Authorization header: %s", err.Error())
	}

	rh.Authorization = authorization
	return nil
}

func parseAuthorizationCredentials(data string) (AuthorizationCredentials, error) {
	credentials := AuthorizationCredentials{}
	parts := splitAuthorizationCredentials(data)

	scheme := lws.TrimRight(parts[0])
	err := constructs.ValidateToken(scheme)
	if err != nil {
		return credentials, fmt.Errorf("malformed Authorization scheme (%s)", data)
	}
	credentials.Scheme = scheme

	err = credentials.setParams(parts[1])
	return credentials, err
}

func splitAuthorizationCredentials(data string) []string {
	i := 0

	for i < len(data) && !constructs.HttpByte(data[i]).IsTSpecial() {
		isNewLineLws, next, _ := lws.NewLine(string(data), i)
		if isNewLineLws {
			i = next
		} else {
			i++
		}
	}

	return []string{string(data[:i]), string(data[min(len(data), i+1):])}
}

func (ac *AuthorizationCredentials) setParams(data string) error {
	params := make(map[string]string)

	if ac.Scheme == "Basic" {
		err := ac.setBasicSchemeParams(data)
		return err
	}

	for i, param := range rules.Extract(data) {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid auth parameter (param %d [%s])", i, data)
		}

		key := parts[0]
		err := constructs.ValidateToken(key)
		if err != nil {
			return fmt.Errorf("invalid auth parameter (param %d [%s])", i, data)
		}

		val, err := constructs.ParseQuotedString(parts[1])
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
	err = constructs.ValidateToken(userid)
	if err != nil && len(userid) > 0 {
		return fmt.Errorf("invalid credentials")
	}

	password := parts[1]
	err = constructs.ValidateText(password)
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
	date, err := constructs.ParseDate(data)
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
			c, next, err := extractComment(data, i)
			if err != nil {
				return fmt.Errorf("Invalid User-Agent header: bad comment - %s", err.Error())
			}

			err = constructs.ValidateComment(c)
			if err != nil {
				return fmt.Errorf("Invalid User-Agent header: bad comment - %s", err.Error())
			}

			userAgent.Comments = append(userAgent.Comments, c)
			i = next

		} else {
			token, next := extractProductVersion(data, i)
			product, err := parseProductVersion(token)
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

func extractComment(data string, start int) (string, int, error) {
	if data[start] != '(' {
		return "", 0, fmt.Errorf("comment must begin with open parenthesis (%s)", data)
	}

	score := 1
	i := start + 1
	for i < len(data) && score > 0 {
		if data[i] == '(' {
			score++
		}
		if data[i] == ')' {
			score--
		}
		i++
	}

	if score > 0 {
		return "", 0, fmt.Errorf("comment not properly closed (%s)", data)
	}

	comment := string(data[start:i])
	isLws, next := lws.Check(data, i)
	for isLws {
		i = next
		isLws, next = lws.Check(data, i)
	}

	return comment, i, nil
}

func extractProductVersion(data string, start int) (string, int) {
	i := start
	isLws, next := lws.Check(data, i)

	for i < len(data) && data[i] != '(' && !isLws {
		i++
		isLws, next = lws.Check(data, i)
	}

	productToken := string(data[start:i])
	for isLws {
		i = next
		isLws, next = lws.Check(data, i)
	}

	return productToken, i

}

func parseProductVersion(data string) (ProductVersion, error) {
	product := ProductVersion{}
	parts := strings.Split(data, "/")
	if len(parts) > 2 {
		return product, fmt.Errorf("product token can only contain up to 1 forward slash (%s)", data)
	}

	err := constructs.ValidateToken(parts[0])
	if err != nil {
		return product, fmt.Errorf("invalid product token (%s)", data)
	}
	product.Product = parts[0]

	if len(parts) == 2 {
		err := constructs.ValidateToken(parts[1])
		if err != nil {
			return product, fmt.Errorf("invalid product token (%s)", data)
		}
		product.Version = parts[1]
	}

	return product, nil

}

func (rh *RequestHeaders) setAllow(data string) error {
	var methods []Method
	rules := rules.Extract(data)
	if len(rules) == 0 {
		return fmt.Errorf("Invalid Allow header: must include at least one method (%s)", data)
	}

	for _, m := range rules {
		err := constructs.ValidateToken(m)

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
	err := constructs.ValidateToken(data)
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
	contentType, err := parseContentType(data)
	if err != nil {
		return fmt.Errorf("Invalid Content-Type header: %s", err.Error())
	}

	rh.ContentType = contentType
	return nil
}

func parseContentType(data string) (ContentType, error) {
	contentType := ContentType{}
	parts := strings.SplitN(data, ";", 2)

	mediaType := strings.Split(lws.Trim(parts[0]), "/")
	if len(mediaType) != 2 {
		return contentType, fmt.Errorf("malformed media type header (%s)", data)
	}

	err := constructs.ValidateToken(mediaType[0])
	if err != nil {
		return contentType, fmt.Errorf("malformed media type (%s)", data)
	}
	contentType.Type = mediaType[0]

	err = constructs.ValidateToken(mediaType[1])
	if err != nil {
		return contentType, fmt.Errorf("malformed media subtype (%s)", data)
	}
	contentType.Subtype = mediaType[1]

	if len(parts) == 2 {
		params, err := parseContentTypeParameters(parts[1])
		if err != nil {
			return contentType, err
		}
		contentType.Parameters = params
	}

	return contentType, nil

}

func parseContentTypeParameters(data string) (map[string]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("parameter cannot be empty (%s)", data)
	}
	parameters := make(map[string]string)

	i := 0
	for i < len(data) {
		isLws, next := lws.Check(data, i)
		for isLws {
			i = next
			isLws, next = lws.Check(data, i)
		}

		attribute := []byte{}
		for i < len(data) && data[i] != '=' {
			attribute = append(attribute, data[i])
			i++
		}

		err := constructs.ValidateToken(string(attribute))
		if err != nil {
			return nil, fmt.Errorf("parameter attribute must be a token (%s)", data)
		}
		i++
		if i >= len(data) {
			return nil, fmt.Errorf("parameter has no value (%s)", data)
		}

		var value string
		if data[i] == '"' {
			v := []byte{data[i]}
			i++
			for i < len(data) && data[i] != '"' {
				v = append(v, data[i])
				i++
			}
			if i < len(data) {
				v = append(v, data[i])
				i++
			}

			value, err = constructs.ParseQuotedString(string(v))
			if err != nil {
				return nil, err
			}

			isLws, next = lws.Check(data, i)
			for isLws {
				i = next
				isLws, next = lws.Check(data, i)
			}
		} else {
			v := []byte{}
			for i < len(data) && data[i] != ';' {
				v = append(v, data[i])
				i++
			}

			value = lws.TrimRight(string(v))
			err := constructs.ValidateToken(value)
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
	expires, err := constructs.ParseDate(data)
	if err != nil {
		return fmt.Errorf("Invalid Expires header: %s", err.Error())
	}

	rh.Date = MessageTime{expires}
	return nil

}

func (rh *RequestHeaders) setLastModified(data string) error {
	lastModified, err := constructs.ParseDate(data)
	if err != nil {
		return fmt.Errorf("Invalid Last-Modified header: %s", err.Error())
	}

	rh.LastModified = MessageTime{lastModified}
	return nil
}

func (rh *RequestHeaders) setUnrecognized(name, data string) error {
	err := constructs.ValidateText(data)
	if err != nil {
		return fmt.Errorf("Invalid %s header: %s", name, err.Error())
	}

	if rh.Unrecognized == nil {
		rh.Unrecognized = make(map[string]string)
	}
	rh.Unrecognized[name] = data
	return nil
}

func parseRequestBody(data []byte, rh RequestHeaders) ([]byte, error) {
	var body []byte
	length := rh.ContentLength

	if length > ContentLength(len(data)) {
		return body, ClientError{message: "Content-Length header exceeds body length"}
	}

	for i := range length {
		body = append(body, data[i])
	}

	return decodeRequestBody(body, rh.ContentEncoding)
}

func decodeRequestBody(body []byte, encoding ContentEncoding) ([]byte, error) {
	var res []byte
	var err error
	reader := bytes.NewReader(body)

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
	reader := lzw.NewReader(r, lzw.LSB, 8)
	defer reader.Close()

	return io.ReadAll(reader)
}
