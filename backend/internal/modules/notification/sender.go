package notification

import (
	"fmt"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
)

// SMTPSender sends plain-text email via net/smtp — no templating engine,
// no HTML; a digest is a few lines of text, that's all this needs.
type SMTPSender struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (s *SMTPSender) Send(to, subject, body string) error {
	if s.Host == "" {
		return nil // SMTP not configured — treat as a no-op, not an error
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", s.From, to, subject, body)
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	return smtp.SendMail(s.Host+":"+s.Port, auth, s.From, []string{to}, []byte(msg))
}

// LineNotifySender posts to the LINE Notify API. Each user supplies their
// own token (see Preference.LineNotifyToken) — there's no way to message
// an arbitrary LINE account from a shared org token.
type LineNotifySender struct {
	Client *http.Client
}

func NewLineNotifySender() *LineNotifySender {
	return &LineNotifySender{Client: http.DefaultClient}
}

func (s *LineNotifySender) Send(token, message string) error {
	req, err := http.NewRequest(http.MethodPost, "https://notify-api.line.me/api/notify",
		strings.NewReader(url.Values{"message": {message}}.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("line notify: unexpected status %d", resp.StatusCode)
	}
	return nil
}
