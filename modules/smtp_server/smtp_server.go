package smtp_server

import (
	"context"
	"fmt"
	"github.com/bettercap/bettercap/session"
	"github.com/emersion/go-smtp"
	"io"
	"io/ioutil"
	"time"
)

type SmtpSession struct {
	mod     *SmtpServer
	logname string
}

func (s *SmtpSession) Mail(from string, opts smtp.MailOptions) error {
	s.mod.Info("Mail from: %s", from)
	return nil
}

func (s *SmtpSession) Rcpt(to string) error {
	s.mod.Info("Rcpt to: %s", to)
	return nil
}

func (s *SmtpSession) Data(r io.Reader) error {
	if b, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		if s.mod.outdir == "" {
			s.mod.Info("Data: %s", string(b))
		} else {
			if err := ioutil.WriteFile(s.mod.outdir+s.logname+".data", b, 0600); err != nil {
				s.mod.Warning("error while saving the file: %s", err)
			}

		}
	}
	return nil
}

func (s *SmtpSession) Reset() {}

func (s *SmtpSession) Logout() error {
	return nil
}

type Backend struct {
	mod *SmtpServer
}

func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	fileName := fmt.Sprintf("/%v", time.Now().Unix())
	info := fmt.Sprintf("Username: %s, Password: %s", username, password)
	if bkd.mod.outdir == "" {
		bkd.mod.Info(info)
	} else {
		if err := ioutil.WriteFile(bkd.mod.outdir+fileName+".pass", []byte(info), 0600); err != nil {
			bkd.mod.Warning("error while saving the file: %s", err)
		}
	}
	return &SmtpSession{mod: bkd.mod, logname: fileName}, nil
}

func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	fileName := fmt.Sprintf("/%v", time.Now().Unix())
	return &SmtpSession{mod: bkd.mod, logname: fileName}, nil
}

type SmtpServer struct {
	session.SessionModule
	server *smtp.Server
	outdir string
}

func NewSmtpServer(s *session.Session) *SmtpServer {
	mod := &SmtpServer{
		SessionModule: session.NewSessionModule("smtp.server", s),
	}

	be := &Backend{
		mod: mod,
	}

	mod.server = smtp.NewServer(be)

	mod.AddParam(session.NewIntParameter("smtp.server.port",
		"25",
		"Port to bind the smtp server to."))

	mod.AddParam(session.NewStringParameter("smtp.server.address",
		session.ParamIfaceAddress,
		session.IPv4Validator,
		"Address to bind the smtp server to."))

	mod.AddParam(session.NewStringParameter("smtp.server.outdir",
		"",
		"",
		"If filled, the mails will be saved to this path instead of being logged."))

	mod.AddHandler(session.NewModuleHandler("smtp.server on", "",
		"Start smtpd server.",
		func(args []string) error {
			return mod.Start()
		}))

	mod.AddHandler(session.NewModuleHandler("smtp.server off", "",
		"Stop smtpd server.",
		func(args []string) error {
			return mod.Stop()
		}))

	return mod
}

func (mod *SmtpServer) Name() string {
	return "smtp.server"
}

func (mod *SmtpServer) Description() string {
	return "A simple SMTP server, to intercept email and password."
}

func (mod *SmtpServer) Author() string {
	return "Michael Scherer <misc@zarb.org>"
}

func (mod *SmtpServer) Configure() error {
	var err error
	var port int
	var addr string
	var outdir string

	if err, port = mod.IntParam("smtp.server.port"); err != nil {
		return err
	}

	if err, addr = mod.StringParam("smtp.server.address"); err != nil {
		return err
	}

	mod.server.Addr = fmt.Sprintf("%s:%v", addr, port)

	if err, outdir = mod.StringParam("smtp.server.outdir"); err != nil {
		return err
	}
	mod.outdir = outdir

	return nil
}

func (mod *SmtpServer) Start() error {
	if err := mod.Configure(); err != nil {
		return err
	}

	return mod.SetRunning(true, func() {
		var err error
		mod.Info("starting on SMTP port %s", mod.server.Addr)
		if mod.outdir != "" {
			mod.Info("logging to %s", mod.outdir)
		}
		// FIXME erreur
		if err = mod.server.ListenAndServe(); err != nil {
			mod.Error("%v", err)
			mod.Stop()
		}
	})

}

func (mod *SmtpServer) Stop() error {
	return mod.SetRunning(false, func() {
		// FIXME check the deal with context here
		_, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		mod.server.Close()
	})

}
