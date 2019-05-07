package auth

type Auth interface {
	GetLoginUid() string
	SetLoginUid(uid string)
	IsLogin() bool
}

type auth struct {
	LoginUid string
}

func NewAuth() Auth {
	return &auth{}
}

func (a *auth) GetLoginUid() string {
	return a.LoginUid
}

func (a *auth) SetLoginUid(uid string) {
	a.LoginUid = uid
}

func (a *auth) IsLogin() bool {
	return a.LoginUid != ""
}
