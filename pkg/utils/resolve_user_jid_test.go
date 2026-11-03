package utils

import (
	"errors"
	"reflect"
	"testing"

	whatsmeow_types "go.mau.fi/whatsmeow/types"
)

func TestUserJIDCandidates(t *testing.T) {
	tests := []struct {
		name     string
		jid      whatsmeow_types.JID
		expected []string
	}{
		{
			name:     "Brazilian mobile with DDD >= 31: as given, then without the ninth digit",
			jid:      whatsmeow_types.NewJID("5541998618419", whatsmeow_types.DefaultUserServer),
			expected: []string{"+5541998618419", "+554198618419"},
		},
		{
			name:     "Brazilian mobile with DDD < 31: only as given",
			jid:      whatsmeow_types.NewJID("5511975342445", whatsmeow_types.DefaultUserServer),
			expected: []string{"+5511975342445"},
		},
		{
			name:     "Leading plus is not doubled",
			jid:      whatsmeow_types.NewJID("+5541998618419", whatsmeow_types.DefaultUserServer),
			expected: []string{"+5541998618419", "+554198618419"},
		},
		{
			name:     "LID is not a phone",
			jid:      whatsmeow_types.NewJID("48155592761478", whatsmeow_types.HiddenUserServer),
			expected: nil,
		},
		{
			name:     "Group is not a phone",
			jid:      whatsmeow_types.NewJID("120363399686447841", whatsmeow_types.GroupServer),
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UserJIDCandidates(tt.jid)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("UserJIDCandidates(%s) = %v, expected %v", tt.jid, got, tt.expected)
			}
		})
	}
}

func TestResolveUserJID(t *testing.T) {
	phone := whatsmeow_types.NewJID("5541998618419", whatsmeow_types.DefaultUserServer)
	lid := whatsmeow_types.NewJID("48155592761478", whatsmeow_types.HiddenUserServer)
	found := func(jid whatsmeow_types.JID) whatsmeow_types.IsOnWhatsAppResponse {
		return whatsmeow_types.IsOnWhatsAppResponse{IsIn: true, JID: jid}
	}
	notFound := whatsmeow_types.IsOnWhatsAppResponse{IsIn: false}

	t.Run("Resolves to the JID WhatsApp answers with (the LID)", func(t *testing.T) {
		var asked []string
		check := func(phones []string) ([]whatsmeow_types.IsOnWhatsAppResponse, error) {
			asked = append(asked, phones...)
			return []whatsmeow_types.IsOnWhatsAppResponse{found(lid)}, nil
		}
		if got := ResolveUserJID(phone, check, nil); got != lid {
			t.Errorf("got %s, expected %s", got, lid)
		}
		if !reflect.DeepEqual(asked, []string{"+5541998618419"}) {
			t.Errorf("asked %v, expected only the number as given", asked)
		}
	})

	t.Run("Falls back to the normalized number when the first form is unknown", func(t *testing.T) {
		var asked []string
		check := func(phones []string) ([]whatsmeow_types.IsOnWhatsAppResponse, error) {
			asked = append(asked, phones...)
			if phones[0] == "+554198618419" {
				return []whatsmeow_types.IsOnWhatsAppResponse{found(lid)}, nil
			}
			return []whatsmeow_types.IsOnWhatsAppResponse{notFound}, nil
		}
		if got := ResolveUserJID(phone, check, nil); got != lid {
			t.Errorf("got %s, expected %s", got, lid)
		}
		if !reflect.DeepEqual(asked, []string{"+5541998618419", "+554198618419"}) {
			t.Errorf("asked %v", asked)
		}
	})

	t.Run("Unknown everywhere: the JID as given", func(t *testing.T) {
		check := func([]string) ([]whatsmeow_types.IsOnWhatsAppResponse, error) {
			return []whatsmeow_types.IsOnWhatsAppResponse{notFound}, nil
		}
		if got := ResolveUserJID(phone, check, nil); got != phone {
			t.Errorf("got %s, expected %s", got, phone)
		}
	})

	t.Run("Query failure: the JID as given, and a warning", func(t *testing.T) {
		warned := 0
		check := func([]string) ([]whatsmeow_types.IsOnWhatsAppResponse, error) {
			return nil, errors.New("info query timed out")
		}
		warn := func(string, ...any) { warned++ }
		if got := ResolveUserJID(phone, check, warn); got != phone {
			t.Errorf("got %s, expected %s", got, phone)
		}
		if warned != 1 {
			t.Errorf("warned %d times, expected 1", warned)
		}
	})

	t.Run("A LID is used as given without asking", func(t *testing.T) {
		check := func([]string) ([]whatsmeow_types.IsOnWhatsAppResponse, error) {
			t.Fatal("must not query WhatsApp for a LID")
			return nil, nil
		}
		if got := ResolveUserJID(lid, check, nil); got != lid {
			t.Errorf("got %s, expected %s", got, lid)
		}
	})
}
