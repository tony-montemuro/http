package message

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
	Location        AbsPathUri //fix later
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

func (r response) Marshal() []byte {
	var marshaled []byte

	line := r.code.marshal()
	marshaled = append(marshaled, line...)

	headers := r.headers.marshal(len(r.body) > 0)
	marshaled = append(marshaled, headers...)

	marshaled = append(marshaled, r.body...)
	return marshaled
}
