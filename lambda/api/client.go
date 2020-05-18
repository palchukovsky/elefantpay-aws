package api

// ClientRequest describes client request.
type ClientRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
