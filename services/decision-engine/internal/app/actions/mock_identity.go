package actions

import (
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/session"
)

func mockIdentitySeed(sess *session.Session) string {
	if sess == nil {
		return "anonymous"
	}

	switch {
	case sess.ID.String() != "00000000-0000-0000-0000-000000000000":
		return sess.ID.String()
	case sess.ExternalUserID != "":
		return fmt.Sprintf("%s:%s", sess.Channel, sess.ExternalUserID)
	case sess.ClientID != "":
		return fmt.Sprintf("%s:%s", sess.Channel, sess.ClientID)
	case sess.UserID.String() != "00000000-0000-0000-0000-000000000000":
		return sess.UserID.String()
	default:
		return "anonymous"
	}
}
