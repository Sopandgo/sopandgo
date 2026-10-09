package mail

import "testing"

func TestEffectiveMailMode(t *testing.T) {
	cases := []struct {
		name                   string
		mode, transport        string
		smtpReady, resendReady bool
		want                   string
	}{
		{"manual stays manual", MailModeManualLinks, MailTransportSMTP, true, true, MailModeManualLinks},
		{"smtp saved", MailModeSMTP, MailTransportSMTP, true, false, MailModeSMTP},
		{"smtp not saved falls back", MailModeSMTP, MailTransportSMTP, false, true, MailModeManualLinks},
		{"resend saved", MailModeSMTP, MailTransportResend, false, true, MailModeSMTP},
		{"resend not saved falls back", MailModeSMTP, MailTransportResend, true, false, MailModeManualLinks},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectiveMailMode(tc.mode, tc.transport, tc.smtpReady, tc.resendReady); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
