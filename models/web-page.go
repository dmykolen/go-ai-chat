package models

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gitlab.dev.ict/golang/libs/utils"
)

const (
	T1 = "AI Web Interface"
	T2 = "VoIP"
	T3 = "AI DB Assistant"

	fmtForUriHST     = "/api/v1/users/%s/chats/%s"
	fmtForUriHST_new = "/api/v1/users//chats/%s"
)

type BasicPageProps struct {
	Title         string
	Login         string
	UriBtnSend    string
	UriBtnHistory string
	Authenticated bool
	EnvDev        bool
}

func NewBpp(t string) *BasicPageProps {
	return &BasicPageProps{Title: t}
}

func (p *BasicPageProps) String() string {
	return utils.JsonPrettyStr(p)
}

func (p *BasicPageProps) Eval() *BasicPageProps {
	if p.UriBtnSend == "" && p.Title != "" {
		switch strings.Trim(p.Title, " ") {
		case T1:
			p.UriBtnSend = "/chatgpt"
		case T2:
			p.UriBtnSend = "/api/v1/ask-ai-voip"
		case T3:
			p.UriBtnSend = "/api/v1/ask-db"
		}
	}

	if p.UriBtnHistory == "" && p.Title != "" {
		// ln := p.Login
		ln := ""
		switch strings.Trim(p.Title, " ") {
		case T1:
			p.UriBtnHistory = fmt.Sprintf(fmtForUriHST, ln, "shortinfo-chatgpt")
		case T2:
			p.UriBtnHistory = fmt.Sprintf(fmtForUriHST, ln, "shortinfo")
		case T3:
			p.UriBtnHistory = fmt.Sprintf(fmtForUriHST, ln, "shortinfo-db-chain")
		}
	}
	return p
}

// Add this method to web-page.go
func (p *BasicPageProps) ToMap() fiber.Map {
	return fiber.Map{
		"Title":         p.Title,
		"Login":         p.Login,
		"UriBtnSend":    p.UriBtnSend,
		"UriBtnHistory": p.UriBtnHistory,
		"Authenticated": p.Authenticated,
		"EnvDev":        p.EnvDev,
	}
}

func (p *BasicPageProps) Auth(a bool) *BasicPageProps {
	p.Authenticated = a
	return p
}

func (p *BasicPageProps) ULogin(l string) *BasicPageProps {
	p.Login = l
	return p
}

func (p *BasicPageProps) Dev(e bool) *BasicPageProps {
	p.EnvDev = e
	return p
}
