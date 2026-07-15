package http

import "github.com/nelsonwerd/countershape/internal/adapters/http/model"

type HTTPRequestWire = model.HTTPRequestWire
type HTTPResponseHeader = model.HTTPResponseHeader
type HTTPResponse = model.HTTPResponse

const HTTPWireProfileV1 = model.HTTPWireProfileV1

func EncodeRequest(stimulus model.HTTPStimulus, port int) (HTTPRequestWire, error) {
	return model.EncodeRequest(stimulus, port)
}

func ParseResponse(raw []byte, policy model.HTTPCapturePolicy) (HTTPResponse, error) {
	return model.ParseResponse(raw, policy)
}
