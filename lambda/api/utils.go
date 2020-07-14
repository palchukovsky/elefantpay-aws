package api

import (
	"fmt"

	"github.com/palchukovsky/elefantpay-aws/elefant"
)

func fmtTransLog(trans *elefant.Trans) string {
	result := fmt.Sprintf(`Trans "%s" "%s" (%d): "%s"(%s, %s) -> %f -> "%s"/"%s"`,
		trans.ID, trans.Status.String(), trans.Status, trans.Method.GetID(),
		trans.Method.GetTypeName(), trans.Method.GetName(), trans.Value,
		trans.Account.GetClientID(), trans.Account.GetID())
	if trans.StatusReason != nil {
		result += fmt.Sprintf(` (%s)`, *trans.StatusReason)
	}
	result += "."
	return result
}
