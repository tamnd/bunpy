package bunpy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"sync"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

type smtpConfig struct {
	host     string
	port     int
	username string
	password string
}

var (
	smtpMu  sync.RWMutex
	smtpCfg smtpConfig
)

func BuildEmail(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.email", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("configure", &goipyObject.BuiltinFunc{
		Name: "configure",
		Call: func(_ any, _ []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			cfg := smtpConfig{port: 587}
			if kwargs != nil {
				if v, ok := kwargs.GetStr("host"); ok {
					cfg.host = v.(*goipyObject.Str).V
				}
				if v, ok := kwargs.GetStr("port"); ok {
					if iv, ok2 := v.(*goipyObject.Int); ok2 {
						cfg.port = int(iv.Int64())
					}
				}
				if v, ok := kwargs.GetStr("username"); ok {
					cfg.username = v.(*goipyObject.Str).V
				}
				if v, ok := kwargs.GetStr("password"); ok {
					cfg.password = v.(*goipyObject.Str).V
				}
			}
			smtpMu.Lock()
			smtpCfg = cfg
			smtpMu.Unlock()
			return goipyObject.None, nil
		},
	})

	mod.Dict.SetStr("send", &goipyObject.BuiltinFunc{
		Name: "send",
		Call: func(_ any, _ []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			cfg := loadSMTPConfig()
			if cfg.host == "" {
				return nil, fmt.Errorf("email.send(): no SMTP host configured (call configure() or set SMTP_HOST)")
			}

			var to []string
			var subject, body, html, from string

			if kwargs != nil {
				if v, ok := kwargs.GetStr("to"); ok {
					switch tv := v.(type) {
					case *goipyObject.Str:
						to = []string{tv.V}
					case *goipyObject.List:
						for _, item := range tv.V {
							if s, ok2 := item.(*goipyObject.Str); ok2 {
								to = append(to, s.V)
							}
						}
					}
				}
				if v, ok := kwargs.GetStr("subject"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						subject = s.V
					}
				}
				if v, ok := kwargs.GetStr("body"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						body = s.V
					}
				}
				if v, ok := kwargs.GetStr("html"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						html = s.V
					}
				}
				if v, ok := kwargs.GetStr("from_"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						from = s.V
					}
				}
			}
			if from == "" {
				from = cfg.username
			}
			if len(to) == 0 {
				return nil, fmt.Errorf("email.send(): 'to' is required")
			}

			msg := buildMIME(from, to, subject, body, html)
			return goipyObject.None, smtpSend(cfg, from, to, msg)
		},
	})

	return mod
}

func loadSMTPConfig() smtpConfig {
	smtpMu.RLock()
	cfg := smtpCfg
	smtpMu.RUnlock()

	if cfg.host == "" {
		cfg.host = os.Getenv("SMTP_HOST")
	}
	if cfg.port == 0 {
		if p := os.Getenv("SMTP_PORT"); p != "" {
			n, _ := strconv.Atoi(p)
			cfg.port = n
		}
		if cfg.port == 0 {
			cfg.port = 587
		}
	}
	if cfg.username == "" {
		cfg.username = os.Getenv("SMTP_USERNAME")
	}
	if cfg.password == "" {
		cfg.password = os.Getenv("SMTP_PASSWORD")
	}
	return cfg
}

func buildMIME(from string, to []string, subject, body, html string) []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(to, ", "))
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")

	if body != "" && html != "" {
		// multipart/alternative
		mw := multipart.NewWriter(&buf)
		fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", mw.Boundary())
		// re-create after header is written
		buf.Reset()
		fmt.Fprintf(&buf, "From: %s\r\n", from)
		fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(to, ", "))
		fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
		fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
		mw2 := multipart.NewWriter(&buf)
		fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", mw2.Boundary())

		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/plain; charset=UTF-8")
		h.Set("Content-Transfer-Encoding", "quoted-printable")
		pw, _ := mw2.CreatePart(h)
		qw := quotedprintable.NewWriter(pw)
		qw.Write([]byte(body))
		qw.Close()

		h2 := make(textproto.MIMEHeader)
		h2.Set("Content-Type", "text/html; charset=UTF-8")
		h2.Set("Content-Transfer-Encoding", "quoted-printable")
		pw2, _ := mw2.CreatePart(h2)
		qw2 := quotedprintable.NewWriter(pw2)
		qw2.Write([]byte(html))
		qw2.Close()

		mw2.Close()
	} else if html != "" {
		fmt.Fprintf(&buf, "Content-Type: text/html; charset=UTF-8\r\n\r\n")
		fmt.Fprintf(&buf, "%s", html)
	} else {
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		fmt.Fprintf(&buf, "%s", body)
	}

	return buf.Bytes()
}

func smtpSend(cfg smtpConfig, from string, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", cfg.host, cfg.port)
	auth := smtp.PlainAuth("", cfg.username, cfg.password, cfg.host)

	if cfg.port == 465 {
		// implicit TLS
		tlsCfg := &tls.Config{ServerName: cfg.host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("email.send(): TLS dial: %w", err)
		}
		defer conn.Close()
		c, err := smtp.NewClient(conn, cfg.host)
		if err != nil {
			return fmt.Errorf("email.send(): SMTP client: %w", err)
		}
		if err2 := c.Auth(auth); err2 != nil {
			return fmt.Errorf("email.send(): auth: %w", err2)
		}
		if err3 := c.Mail(from); err3 != nil {
			return err3
		}
		for _, r := range to {
			if err4 := c.Rcpt(r); err4 != nil {
				return err4
			}
		}
		w, err5 := c.Data()
		if err5 != nil {
			return err5
		}
		w.Write(msg)
		w.Close()
		return c.Quit()
	}

	return smtp.SendMail(addr, auth, from, to, msg)
}

// SMTPDialer is exported for test injection. If non-nil, used instead of real SMTP.
var SMTPDialer func(addr string) (net.Conn, error)
