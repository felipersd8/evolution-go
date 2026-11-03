package utils

import (
	"strings"

	whatsmeow_types "go.mau.fi/whatsmeow/types"
)

// IsOnWhatsAppFunc is whatsmeow's Client.IsOnWhatsApp with the context bound:
// phone numbers in international format, "+" included.
type IsOnWhatsAppFunc func(phones []string) ([]whatsmeow_types.IsOnWhatsAppResponse, error)

// UserJIDCandidates lists the phone forms to ask WhatsApp about for a user
// JID, in order: the number as given, then the CreateJID-normalized one (for
// Brazil that drops the ninth digit on DDD >= 31). Both carry the "+" that
// IsOnWhatsApp expects. Non-user JIDs (group, broadcast, newsletter, LID)
// produce no candidates.
func UserJIDCandidates(jid whatsmeow_types.JID) []string {
	if jid.Server != whatsmeow_types.DefaultUserServer {
		return nil
	}
	user := strings.TrimPrefix(jid.User, "+")
	if user == "" {
		return nil
	}
	candidates := []string{"+" + user}
	normalized, err := CreateJID(user)
	if err != nil {
		return candidates
	}
	normalized = strings.TrimSuffix(normalized, "@"+whatsmeow_types.DefaultUserServer)
	if strings.HasPrefix(normalized, "+") && normalized != candidates[0] {
		candidates = append(candidates, normalized)
	}
	return candidates
}

// ResolveUserJID returns the JID WhatsApp addresses the user by — the LID, or
// the canonical phone JID — through the same IsOnWhatsApp lookup the send
// path does (sendService.validateAndCheckUserExists), so an edit or a revoke
// references the chat the original message went to.
//
// Without it, whatsmeow maps the phone JID to a LID on its own, through
// GetUserInfo; for many numbers that query answers without a LID or never
// answers, and the edit dies with "recipient is not on WhatsApp: no LID found"
// while the message itself had gone out fine (three cases between 27/10 and
// 03/11/2026).
//
// Non-user JIDs, a failed query and an unknown number all come back as given:
// the caller then sends to what it was asked, like the send path does when its
// check fails.
func ResolveUserJID(jid whatsmeow_types.JID, check IsOnWhatsAppFunc, warn func(format string, args ...any)) whatsmeow_types.JID {
	for _, phone := range UserJIDCandidates(jid) {
		resp, err := check([]string{phone})
		if err != nil {
			if warn != nil {
				warn("could not resolve %s on WhatsApp (%v); using it as given", jid, err)
			}
			return jid
		}
		if len(resp) > 0 && resp[0].IsIn && !resp[0].JID.IsEmpty() {
			return resp[0].JID
		}
	}
	return jid
}
