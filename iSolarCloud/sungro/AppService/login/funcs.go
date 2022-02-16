package login

import (
	"GoSungro/Only"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)


type SunGroAuth struct {
	TokenExpiry string

	AppKey      string
	Username    string
	Password    string
}

type Token struct {
	EndPoint

	TokenFile   string
	TokenExpiry time.Time
	newToken    bool
	retry       int
}


func (e *EndPoint) Login(auth *SunGroAuth) error {
	for range Only.Once {
		_ = e.readTokenFile()

		e.Error = e.Verify(auth)
		if e.Error != nil {
			break
		}

		e.Error = e.RetrieveToken()
		if e.Error != nil {
			break
		}
	}

	return e.Error
}

func (e *EndPoint) GetToken() string {
	return e.GetResponse().ResultData.Token
}

func (e *EndPoint) Verify(auth *SunGroAuth) error {
	for range Only.Once {
		if auth == nil {
			// If nil, then assume we haven'e set anything.
			break
		}

		if auth.AppKey == "" {
			e.Error = errors.New("API AppKey")
			break
		}
		if auth.Username == "" {
			e.Error = errors.New("empty Client ApiUsername")
			break
		}
		if auth.Password == "" {
			e.Error = errors.New("empty Client ApiPassword")
			break
		}

		e.Token = e.Response.(Response).ResultData.Token
		if e.Response.ResultData.Token == "" {
			e.newToken = true
		}

		if auth.TokenExpiry == "" {
			auth.TokenExpiry = time.Now().Format(DateTimeFormat)
		}
		e.TokenExpiry, e.Error = time.Parse(DateTimeFormat, auth.TokenExpiry)
		if e.Error != nil {
			e.newToken = true
		}

		e.Request = login.Request {
			Appkey:   auth.AppKey,
			SysCode:  "900",
			UserAccount: auth.Username,
			UserPassword: auth.Password,
		}

		e.HasTokenExpired()
	}

	return e.Error
}

func (e *EndPoint) RetrieveToken() error {
	for range Only.Once {
		e.HasTokenExpired()
		if !e.newToken {
			break
		}

		u := fmt.Sprintf("%s%s",
			e.Url.String(),
			TokenRequestUrl,
		)
		//p, _ := json.Marshal(map[string]string {
		//	"user_account": e.Request.Username,
		//	"user_password": e.Request.Password,
		//	"appkey": e.Request.AppKey,
		//	"sys_code": "900",
		//})
		p, _ := json.Marshal(e.Request)

		var response *http.Response
		response, e.Error = http.Post(u, "application/json", bytes.NewBuffer(p))
		if e.Error != nil {
			break
		}
		//goland:noinspection GoUnhandledErrorResult
		defer response.Body.Close()
		if response.StatusCode != 200 {
			e.Error = errors.New(fmt.Sprintf("Status Code is %d", response.StatusCode))
			break
		}

		var body []byte
		body, e.Error = ioutil.ReadAll(response.Body)
		if e.Error != nil {
			break
		}

		e.Error = json.Unmarshal(body, &e.Response)
		if e.Error != nil {
			break
		}

		e.TokenExpiry = time.Now()

		e.Error = e.saveToken()
		if e.Error != nil {
			break
		}
	}

	return e.Error
}

func (e *EndPoint) HasTokenExpired() bool {
	for range Only.Once {
		if e.TokenExpiry.Before(time.Now()) {
			e.newToken = true
			break
		}

		if e.Response.ResultData.Token == "" {
			e.newToken = true
			break
		}
	}

	return e.newToken
}

func (e *EndPoint) HasTokenChanged() bool {
	ok := e.newToken
	if e.newToken {
		e.newToken = false
	}
	return ok
}

func (e *EndPoint) GetTokenExpiry() time.Time {
	return e.TokenExpiry
}

// Retrieves a token from a local file.
func (e *EndPoint) readTokenFile() error {
	for range Only.Once {
		if e.TokenFile == "" {
			e.TokenFile, e.Error = os.UserHomeDir()
			if e.Error != nil {
				e.TokenFile = ""
				break
			}
			e.TokenFile = filepath.Join(e.TokenFile, ".GoSungro", DefaultAuthTokenFile)
		}

		var f *os.File
		f, e.Error = os.Open(e.TokenFile)
		if e.Error != nil {
			break
		}

		//goland:noinspection GoUnhandledErrorResult
		defer f.Close()
		//ret = &oauth2.token{}
		e.Error = json.NewDecoder(f).Decode(&e.Response)
	}

	return e.Error
}

// Saves a token to a file path.
func (e *EndPoint) saveToken() error {
	for range Only.Once {
		if e.TokenFile == "" {
			e.TokenFile, e.Error = os.UserHomeDir()
			if e.Error != nil {
				e.TokenFile = ""
				break
			}
			e.TokenFile = filepath.Join(e.TokenFile, ".GoSungro", DefaultAuthTokenFile)
		}

		fmt.Printf("Saving token file to: %s\n", e.TokenFile)
		var f *os.File
		f, e.Error = os.OpenFile(e.TokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if e.Error != nil {
			e.Error = errors.New(fmt.Sprintf("Unable to cache SUNGRO oauth token: %v", e.Error))
			break
		}

		//goland:noinspection GoUnhandledErrorResult
		defer f.Close()
		e.Error = json.NewEncoder(f).Encode(e.Response.ResultData)
	}

	return e.Error
}