package api

// AuthTokenHeaderName is the name of an auth_token header.
const AuthTokenHeaderName = "Auth-Token"

func isDev(request *httpRequest) bool {
	return request.RequestContext.Stage == "dev"
}
