package beeper

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RespStartLogin struct {
	RequestID string    `json:"request"`
	Type      []string  `json:"type"`
	Expires   time.Time `json:"expires"`
}

type ReqSendLoginEmail struct {
	RequestID string `json:"request"`
	Email     string `json:"email"`
}

type ReqSendLoginCode struct {
	RequestID string `json:"request"`
	Code      string `json:"response"`
}

type RespSendLoginCode struct {
	LoginToken string `json:"token"`
}

var ErrInvalidLoginCode = fmt.Errorf("invalid login code")

const loginAuth = "BEEPER-PRIVATE-API-PLEASE-DONT-USE"

func StartLogin() (resp *RespStartLogin, err error) {
	req := newRequest(loginAuth, http.MethodPost, "/user/login")
	req.Body = io.NopCloser(bytes.NewReader([]byte("{}")))
	err = doRequest(req, nil, &resp)
	return
}

func SendLoginEmail(request, email string) error {
	req := newRequest(loginAuth, http.MethodPost, "/user/login/email")
	reqData := &ReqSendLoginEmail{
		RequestID: request,
		Email:     email,
	}
	return doRequest(req, reqData, nil)
}

func SendLoginCode(request, code string) (resp *RespSendLoginCode, err error) {
	req := newRequest(loginAuth, http.MethodPost, "/user/login/response")
	reqData := &ReqSendLoginCode{
		RequestID: request,
		Code:      code,
	}
	err = doRequest(req, reqData, &resp)
	return
}
