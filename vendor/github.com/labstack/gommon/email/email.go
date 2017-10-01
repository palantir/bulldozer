package email

import (
	"bytes"
	"crypto/tls"
	"html/template"
	"net/mail"
	"net/smtp"

	"net"

	"time"

	"github.com/labstack/gommon/random"
)

type (
	Email struct {
		Auth        smtp.Auth
		Header      map[string]string
		Template    *template.Template
		smtpAddress string
	}

	Message struct {
		ID          string  `json:"id"`
		From        string  `json:"from"`
		To          string  `json:"to"`
		CC          string  `json:"cc"`
		Subject     string  `json:"subject"`
		Text        string  `json:"text"`
		HTML        string  `json:"html"`
		Inlines     []*File `json:"inlines"`
		Attachments []*File `json:"attachments"`
		buffer      *bytes.Buffer
		boundary    string
	}

	File struct {
		Name    string
		Type    string
		Content string
	}
)

func New(smtpAddress string) *Email {
	return &Email{
		smtpAddress: smtpAddress,
		Header:      map[string]string{},
	}
}

func (m *Message) writeHeader(key, value string) {
	m.buffer.WriteString(key)
	m.buffer.WriteString(": ")
	m.buffer.WriteString(value)
	m.buffer.WriteString("\r\n")
}

func (m *Message) writeBoundary() {
	m.buffer.WriteString("--")
	m.buffer.WriteString(m.boundary)
	m.buffer.WriteString("\r\n")
}

func (m *Message) writeText(content string, contentType string) {
	m.writeBoundary()
	m.writeHeader("Content-Type", contentType+"; charset=UTF-8")
	m.buffer.WriteString("\r\n")
	m.buffer.WriteString(content)
	m.buffer.WriteString("\r\n")
	m.buffer.WriteString("\r\n")
}

func (m *Message) writeFile(f *File, disposition string) {
	m.writeBoundary()
	m.writeHeader("Content-Type", f.Type+`; name="`+f.Name+`"`)
	m.writeHeader("Content-Disposition", disposition+`; filename="`+f.Name+`"`)
	m.writeHeader("Content-Transfer-Encoding", "base64")
	m.buffer.WriteString("\r\n")
	m.buffer.WriteString(f.Content)
	m.buffer.WriteString("\r\n")
	m.buffer.WriteString("\r\n")
}

func (e *Email) Send(m *Message) (err error) {
	// Message header
	m.buffer = bytes.NewBuffer(make([]byte, 256))
	m.buffer.Reset()
	m.boundary = random.String(16)
	m.writeHeader("MIME-Version", "1.0")
	m.writeHeader("Message-ID", m.ID)
	m.writeHeader("Date", time.Now().Format(time.RFC1123Z))
	m.writeHeader("From", m.From)
	m.writeHeader("To", m.To)
	if m.CC != "" {
		m.writeHeader("CC", m.CC)
	}
	if m.Subject != "" {
		m.writeHeader("Subject", m.Subject)
	}
	// Extra
	for k, v := range e.Header {
		m.writeHeader(k, v)
	}
	m.writeHeader("Content-Type", "multipart/mixed; boundary="+m.boundary)
	m.buffer.WriteString("\r\n")

	// Message body
	if m.Text != "" {
		m.writeText(m.Text, "text/plain")
	} else if m.HTML != "" {
		m.writeText(m.HTML, "text/html")
	} else {
		// TODO:
	}

	// Inlines/attachments
	for _, f := range m.Inlines {
		m.writeFile(f, "inline")
	}
	for _, f := range m.Attachments {
		m.writeFile(f, "attachment")
	}
	m.buffer.WriteString("--")
	m.buffer.WriteString(m.boundary)
	m.buffer.WriteString("--")

	// Dial
	c, err := smtp.Dial(e.smtpAddress)
	if err != nil {
		return
	}
	defer c.Close()

	// Check if TLS is required
	if ok, _ := c.Extension("STARTTLS"); ok {
		host, _, _ := net.SplitHostPort(e.smtpAddress)
		config := &tls.Config{ServerName: host}
		if err = c.StartTLS(config); err != nil {
			return err
		}
	}

	// Authenticate
	if e.Auth != nil {
		if err = c.Auth(e.Auth); err != nil {
			return
		}
	}

	// Send message
	from, err := mail.ParseAddress(m.From)
	if err != nil {
		return
	}
	if err = c.Mail(from.Address); err != nil {
		return
	}
	to, err := mail.ParseAddressList(m.To)
	if err != nil {
		return
	}
	for _, a := range to {
		if err = c.Rcpt(a.Address); err != nil {
			return
		}
	}
	wc, err := c.Data()
	if err != nil {
		return
	}
	defer wc.Close()
	_, err = m.buffer.WriteTo(wc)
	return
}
