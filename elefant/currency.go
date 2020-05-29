package elefant

// Currency describes currency interface.
type Currency interface{ GetISO() string }

// NewCurrency creates new currency instance with given ISO code.
func NewCurrency(iso string) Currency { return &currency{iso: iso} }

type currency struct{ iso string }

func (currency *currency) GetISO() string { return currency.iso }
