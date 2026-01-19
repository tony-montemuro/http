package http

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
	Date            MessageTime
	Pragma          PragmaDirectives
	Location        Uri
	Server          server
	WwwAuthenticate challenge
	Allow           Methods
	ContentEncoding ContentEncoding
	ContentLength   ContentLength
	ContentType     ContentType
	Expires         MessageTime
	LastModified    MessageTime
	Unrecognized    map[string]string
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
