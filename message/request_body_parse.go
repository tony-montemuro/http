package message

import (
	"bytes"
	"compress/gzip"
	"compress/lzw"
	"fmt"
	"io"
)

type requestBodyParser []byte

func (rb requestBodyParser) parse(rh RequestHeaders) ([]byte, error) {
	var body []byte
	length := rh.ContentLength

	if length > uint64(len(rb)) {
		return body, ClientError{message: "Content-Length header exceeds body length"}
	}

	for i := range length {
		body = append(body, rb[i])
	}

	return requestBodyDecoder(body).decode()
}

type requestBodyDecoder []byte

func (d requestBodyDecoder) decode() ([]byte, error) {
	var res []byte
	var err error
	reader := bytes.NewReader([]byte(d))

	switch ContentEncoding(d) {
	case ContentEncodingXGzip:
		res, err = gzipDecode(reader)
	case ContentEncodingXCompress:
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
		return nil, fmt.Errorf("unexpected issue decoding body (%w)", err)
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
