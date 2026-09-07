package recovery

import (
	"context"
	"net"
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
		ResetURL: "https://app.example.com/reset-password",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	link, err := n.resetLink("a+b/=")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Query().Get("token"); got != "a+b/=" {
		t.Fatalf("token=%q", got)
	}
	msg := n.message("a@example.com", link, time.Unix(1_700_000_000, 0).UTC())
	if !strings.Contains(msg, "Subject: LinkUp password reset") || !strings.Contains(msg, link) {
		t.Fatal("reset message missing required fields")
	}
}

func TestSMTPConfigRejectsPartialCredentialsHeaderInjectionAndUnsafeResetOrigins(t *testing.T) {
	base := SMTPConfig{Address: "smtp.example.com:587", Host: "smtp.example.com", From: "no-reply@example.com", ResetURL: "https://app.example.com/reset-password"}

	partial := base
	partial.Username = "user"
	if _, err := NewSMTP(partial); err == nil {
		t.Fatal("partial credentials must fail")
	}

	header := base
	header.From = "no-reply@example.com\r\nBcc: attacker@example.com"
	if _, err := NewSMTP(header); err == nil {
		t.Fatal("header injection must fail")
	}

	unsafe := []string{
		"http://app.example.com/reset-password",
		"linkup://reset-password",
		"https://user:pass@app.example.com/reset-password",
		"https://app.example.com/reset-password?next=https://evil.example",
		"https://app.example.com/reset-password#token",
	}
	for _, resetURL := range unsafe {
		cfg := base
		cfg.ResetURL = resetURL
		if _, err := NewSMTP(cfg); err == nil {
			t.Fatalf("unsafe reset URL accepted: %s", resetURL)
		}
	}
}

func TestSMTPFullOperationTimeoutBoundsStalledGreeting(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	accepted := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		close(accepted)
		defer conn.Close()
		<-time.After(2 * time.Second)
	}()

	n, err := NewSMTP(SMTPConfig{
		Address:  listener.Addr().String(),
		Host:     "localhost",
		From:     "no-reply@example.com",
		ResetURL: "https://app.example.com/reset-password",
		Timeout:  75 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	err = n.SendPasswordReset(context.Background(), "person@example.com", "token", time.Now().Add(time.Hour))
	if err == nil {
		t.Fatal("stalled SMTP greeting must time out")
	}
	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("test server did not accept connection")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("SMTP operation exceeded bounded timeout: %s", elapsed)
	}
}

func TestSMTPTimeoutCeiling(t *testing.T) {
	_, err := NewSMTP(SMTPConfig{
		Address:  "smtp.example.com:587",
		Host:     "smtp.example.com",
		From:     "no-reply@example.com",
		ResetURL: "https://app.example.com/reset-password",
		Timeout:  maxSMTPOperationTimeout + time.Second,
	})
	if err == nil {
		t.Fatal("oversized SMTP timeout must fail")
	}
}
