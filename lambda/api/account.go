package api

// AccountRequest describes account request.
type AccountRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
