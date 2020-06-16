package elefant

// SendGridAPIKey is a SendGrid auth token (SENDGRID_API_KEY). Set by builder.
var SendGridAPIKey = ""

// EmailFromName is a sender name for emails. Set by builder.
var EmailFromName = ""

// EmailFromAddress is a sender address for emails. Set by builder.
var EmailFromAddress = ""

// Version is a product version. Set by builder.
var Version = ""

// IsDev returns true if build is not production.
func IsDev() bool { return Version == "dev" }
