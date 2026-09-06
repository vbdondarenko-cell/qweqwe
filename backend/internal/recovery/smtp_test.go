package recovery

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSMTPConfigAndResetLink(t *testing.T) {
	n, err := NewSMTP(SMTPConfig{
		Address:  "smtp.example.com:587",
		Host:     "smtp.example.com",
		From:     "LinkUp <no-reply@example.com>",
		ResetURL: "linkup://reset-password",
		Timeout:  5 * time.Second,
	})
	if err != nil { t.Fatal(err) }
	link, err := n.resetLink("a+b/=")
	if err != nil { t.Fatal(err) }
	parsed, err := url.Parse(link)
	if err != nil { t.Fatal(err) }
	if got := parsed.Query().Get("token"); got != "a+b/=" { t.Fatalf("token=%q", got) }
	msg := n.message("a@example.com", link, time.Unix(1_700_000_000, 0).UTC())
	if !strings.Contains(msg, "Subject: LinkUp password reset") || !strings.Contains(msg, link) { t.Fatal("reset message missing required fields") }
}

func TestSMTPConfigRejectsPartialCredentialsAndHeaderInjection(t *testing.T) {
	_, err := NewSMTP(SMTPConfig{Address:"smtp.example.com:587",Host:"smtp.example.com",From:"no-reply@example.com",ResetURL:"linkup://reset-password",Username:"user"})
	if err == nil { t.Fatal("partial credentials must fail") }
	_, err = NewSMTP(SMTPConfig{Address:"smtp.example.com:587",Host:"smtp.example.com",From:"no-reply@example.com\r\nBcc: attacker@example.com",ResetURL:"linkup://reset-password"})
	if err == nil { t.Fatal("header injection must fail") }
}
