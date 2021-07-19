package main

import (
	"context"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-openapi/runtime/client"
	hydraadmin "github.com/ory/hydra-client-go/client/admin"
	hm "github.com/ory/hydra-client-go/models"
	"github.com/utrack/gin-csrf"
	"golang.org/x/oauth2"
	"net/http"
)

func main() {
	transport := httptransport.New("localhost:4445", "/", []string{"http"})
	hydra := hydraadmin.New(transport, nil)

	router := gin.Default()
	router.LoadHTMLGlob("views/**")

	store := cookie.NewStore(
		[]byte("mysecretmysecretmysecretmysecret"),
		[]byte("encryptionkey123"),
	)
	router.Use(sessions.Sessions("m_sess", store))
	router.Use(csrf.Middleware(csrf.Options{
		Secret:    "secret",
		ErrorFunc: errHandler,
	}))

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	router.GET("/login", func(c *gin.Context) {
		challenge := c.Query("login_challenge")

		if challenge == "" {
			fmt.Printf("failed to login, challenge empty\n")
			errHandler(c)
			return
		}

		loginRequest, err := hydra.GetLoginRequest(&hydraadmin.GetLoginRequestParams{
			LoginChallenge: challenge,
			Context:        context.Background(),
		})
		if err != nil {
			fmt.Printf("failed to GetLoginRequest: %s\n", err.Error())
			errHandler(c)
			return
		}

		body := loginRequest.Payload
		if *body.Skip {
			acceptLogin, err := hydra.AcceptLoginRequest(&hydraadmin.AcceptLoginRequestParams{
				Body: &hm.AcceptLoginRequest{
					Subject: body.Subject,
				},
				LoginChallenge: challenge,
				Context:        c,
			})
			if err != nil {
				fmt.Printf("failed to AcceptLoginRequest: %s\n", err.Error())
				errHandler(c)
				return
			}

			c.Redirect(http.StatusMovedPermanently, *acceptLogin.Payload.RedirectTo)
			return
		}

		c.HTML(http.StatusOK, "login.tmpl.html", gin.H{
			"csrf":      csrf.GetToken(c),
			"challenge": challenge,
		})
	})

	router.POST("/login", func(c *gin.Context) {
		challenge := c.PostForm("_challenge")
		email := c.PostForm("email")
		password := c.PostForm("password")

		// go to kratos and check account exists
		if email != "dan@dsjones.me" || password != "password1" {
			fmt.Printf("username/password incorrect")
			errHandler(c)
			return
		}

		subject := "00000000-0000-0000-0000-000000000000"
		acceptLogin, err := hydra.AcceptLoginRequest(&hydraadmin.AcceptLoginRequestParams{
			Body: &hm.AcceptLoginRequest{
				Subject: &subject,
			},
			LoginChallenge: challenge,
			Context:        c,
		})
		if err != nil {
			fmt.Printf("failed to AcceptLoginRequest: %s\n", err.Error())
			errHandler(c)
			return
		}

		c.Redirect(http.StatusMovedPermanently, *acceptLogin.Payload.RedirectTo)
	})

	router.GET("/consent", func(c *gin.Context) {
		consentChallenge := c.Query("consent_challenge")
		consentResp, err := hydra.GetConsentRequest(&hydraadmin.GetConsentRequestParams{
			ConsentChallenge: consentChallenge,
			Context:          c,
		})
		if err != nil {
			fmt.Printf("failed to AcceptConsentRequest: %s\n", err.Error())
			errHandler(c)
			return
		}
		consent := consentResp.Payload
		if consent.Client.Owner == "dan" {
			acceptConsentResp, err := hydra.AcceptConsentRequest(&hydraadmin.AcceptConsentRequestParams{
				Body: &hm.AcceptConsentRequest{
					GrantAccessTokenAudience: consent.RequestedAccessTokenAudience,
					GrantScope:               consent.RequestedScope,
					Remember:                 true,
					RememberFor:              60 * 60,
				},
				ConsentChallenge: consentChallenge,
				Context:          c,
			})
			if err != nil {
				fmt.Printf("failed to AcceptConsentRequest: %s\n", err.Error())
				errHandler(c)
				return
			}

			c.Redirect(http.StatusMovedPermanently, *acceptConsentResp.Payload.RedirectTo)
			return
		}

		c.HTML(http.StatusOK, "consent.tmpl.html", gin.H{
			"csrf":             csrf.GetToken(c),
			"challenge":        consentChallenge,
			"requested_scopes": consent.RequestedScope,
		})
	})

	router.POST("/consent", func(c *gin.Context) {
		challenge := c.PostForm("_challenge")
		grantScopes := c.PostFormArray("grant_scope")
		remember := c.PostForm("remember") == "1"
		submit := c.PostForm("submit")
		if submit == "Allow" {
			acceptResp, err := hydra.AcceptConsentRequest(&hydraadmin.AcceptConsentRequestParams{
				Body: &hm.AcceptConsentRequest{
					GrantScope: grantScopes,
					Remember:   remember,
				},
				ConsentChallenge: challenge,
				Context:          c,
			})
			if err != nil {
				fmt.Printf("failed to AcceptConsentRequest: %s\n", err.Error())
				errHandler(c)
				return
			}

			c.Redirect(http.StatusMovedPermanently, *acceptResp.Payload.RedirectTo)
			return
		}

		rejectResp, err := hydra.RejectConsentRequest(&hydraadmin.RejectConsentRequestParams{
			ConsentChallenge: challenge,
			Context:          c,
		})
		if err != nil {
			fmt.Printf("failed to RejectConsentRequest: %s\n", err.Error())
			errHandler(c)
			return
		}

		c.Redirect(http.StatusMovedPermanently, *rejectResp.Payload.RedirectTo)
	})

	router.GET("/callback", func(c *gin.Context) {
		code := c.Query("code")
		scope := c.Query("scope")

		client := oauth2.Config{
			ClientID:     "demo",
			ClientSecret: "secret",
			Endpoint: oauth2.Endpoint{
				TokenURL:  "http://localhost:4444/oauth2/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
			RedirectURL: "http://127.0.0.1:8080/callback",
			Scopes:      []string{scope},
		}
		token, err := client.Exchange(c, code)
		if err != nil {
			fmt.Printf("failed to Exchange: %s\n", err.Error())
			errHandler(c)
			return
		}

		c.JSON(http.StatusOK, token)
	})

	if err := router.Run(); err != nil {
		panic(err)
	}
}

func errHandler(c *gin.Context) {
	c.HTML(http.StatusBadRequest, "error.tmpl.html", nil)
}
