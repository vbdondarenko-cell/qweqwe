package recovery

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

type SMTPConfig struct {
	Address     string
	Host        string
	Username    string
	Password    string
	From        string
	ResetURL    string
	ImplicitTLS bool
	Timeout     time.Duration
}

type SMTPNotifier struct {
	cfg      SMTPConfig
	fromAddr string
}

func NewSMTP(cfg SMTPConfig) (*SMTPNotifier, error) {
	cfg.Address = strings.TrimSpace(cfg.Address)
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.From = strings.TrimSpace(cfg.From)
	cfg.ResetURL = strings.TrimSpace(cfg.ResetURL)
	if cfg.Timeout <= 0 { cfg.Timeout = 10 * time.Second }
	if cfg.Address == "" || cfg.Host == "" || cfg.From == "" || cfg.ResetURL == "" {
		return nil, errors.New("incomplete SMTP recovery configuration")
	}
	if strings.ContainsAny(cfg.From, "\r\n") { return nil, errors.New("invalid recovery From header") }
	parsedFrom, err := mail.ParseAddress(cfg.From)
	if err != nil { return nil, fmt.Errorf("invalid recovery From address: %w", err) }
	if _, _, err := net.SplitHostPort(cfg.Address); err != nil { return nil, fmt.Errorf("invalid SMTP address: %w", err) }
	if _, err := url.ParseRequestURI(cfg.ResetURL); err != nil { return nil, fmt.Errorf("invalid recovery reset URL: %w", err) }
	if (cfg.Username == "") != (cfg.Password == "") { return nil, errors.New("SMTP username and password must be configured together") }
	return &SMTPNotifier{cfg: cfg, fromAddr: parsedFrom.Address}, nil
}

func (n *SMTPNotifier) SendPasswordReset(ctx context.Context, recipientEmail, rawToken string, expiresAt time.Time) error {
	recipient, err := mail.ParseAddress(strings.TrimSpace(recipientEmail))
	if err != nil { return errors.New("invalid recovery recipient") }
	resetLink, err := n.resetLink(rawToken)
	if err != nil { return err }

	message := n.message(recipient.Address, resetLink, expiresAt.UTC())
	dialer := &net.Dialer{Timeout: n.cfg.Timeout}
	tlsConfig := &tls.Config{ServerName: n.cfg.Host, MinVersion: tls.VersionTLS12}

	var conn net.Conn
	if n.cfg.ImplicitTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", n.cfg.Address, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", n.cfg.Address)
	}
	if err != nil { return err }
	defer conn.Close()

	client, err := smtp.NewClient(conn, n.cfg.Host)
	if err != nil { return err }
	defer client.Close()

	if !n.cfg.ImplicitTLS {
		ok, _ := client.Extension("STARTTLS")
		if !ok { return errors.New("SMTP server does not offer STARTTLS") }
		if err := client.StartTLS(tlsConfig); err != nil { return err }
	}
	if n.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", n.cfg.Username, n.cfg.Password, n.cfg.Host)); err != nil { return err }
	}
	if err := client.Mail(n.fromAddr); err != nil { return err }
	if err := client.Rcpt(recipient.Address); err != nil { return err }
	writer, err := client.Data()
	if err != nil { return err }
	if _, err := io.WriteString(writer, message); err != nil { _ = writer.Close(); return err }
	if err := writer.Close(); err != nil { return err }
	return client.Quit()
}

func (n *SMTPNotifier) resetLink(rawToken string) (string, error) {
	base, err := url.Parse(n.cfg.ResetURL)
	if err != nil { return "", err }
	query := base.Query()
	query.Set("token", rawToken)
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func (n *SMTPNotifier) message(recipient, resetLink string, expiresAt time.Time) string {
	var b strings.Builder
	w := bufio.NewWriter(&b)
	_, _ = fmt.Fprintf(w, "From: %s\r\n", n.cfg.From)
	_, _ = fmt.Fprintf(w, "To: %s\r\n", recipient)
	_, _ = fmt.Fprint(w, "Subject: LinkUp password reset\r\n")
	_, _ = fmt.Fprint(w, "MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
	_, _ = fmt.Fprintf(w, "A password reset was requested for your LinkUp account.\r\n\r\nOpen this link to choose a new password:\r\n%s\r\n\r\nThis link expires at %s. If you did not request this, ignore this email.\r\n", resetLink, expiresAt.Format(time.RFC3339))
	_ = w.Flush()
	return b.String()
}
