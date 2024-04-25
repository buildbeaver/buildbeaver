package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/middleware"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/services"
	ghub "github.com/buildbeaver/buildbeaver/server/services/scm/github"
)

const (
	sessionName                   = "buildbeaver"
	sessionIdentityIDKeyName      = "identity_id"
	sessionOAuthTokenKeyName      = "oauth_token"
	sessionGitHubAuthStateKeyName = "github_state"

	sessionOAuthRedirectSuccessKeyName = "redirect_success_url"
	sessionOAuthRedirectErrorKeyName   = "redirect_error_url"
	sessionExpirySeconds               = 3600 * 24 * 5 // 5 days
)

// UseSameSiteNoneMode is a setting to set SameSite=none mode when issuing session cookies, so the cookies will
// be sent along with cross-site requests. This should only be used in development environments, not in production.
type UseSameSiteNoneMode bool

func (b UseSameSiteNoneMode) Bool() bool {
	return bool(b)
}

type SessionAuthenticationKey [32]byte
type SessionEncryptionKey [32]byte

type GitHubOAuth2Config = oauth2.Config

type AuthenticationConfig struct {
	SessionAuthenticationKey SessionAuthenticationKey
	SessionEncryptionKey     SessionEncryptionKey
	UseSameSiteNoneMode      UseSameSiteNoneMode
	GitHub                   GitHubOAuth2Config
}

type CoreAuthenticationAPI struct {
	authenticationService services.AuthenticationService
	sessionStore          sessions.Store
	config                AuthenticationConfig
	*APIBase
}

func NewCoreAuthenticationAPI(
	authenticationService services.AuthenticationService,
	authorizationService services.AuthorizationService,
	resourceLinker *routes.ResourceLinker,
	logFactory logger.LogFactory,
	config AuthenticationConfig,
) *CoreAuthenticationAPI {

	sessionStore := sessions.NewCookieStore(
		config.SessionAuthenticationKey[:],
		config.SessionEncryptionKey[:])

	sessionStore.Options.Secure = true

	// In order to support CORS requests in our dev environments in Chrome, we need to send cookies to
	// other sites using SameSite=None mode.
	// For production use SameSite=Lax mode.
	// TODO  This should be strict once we're issuing the cookie on the correct domain in prod
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite
	// https://blog.heroku.com/chrome-changes-samesite-cookie
	if config.UseSameSiteNoneMode {
		sessionStore.Options.SameSite = http.SameSiteNoneMode
	} else {
		sessionStore.Options.SameSite = http.SameSiteLaxMode
	}

	return &CoreAuthenticationAPI{
		authenticationService: authenticationService,
		sessionStore:          sessionStore,
		config:                config,
		APIBase:               NewAPIBase(authorizationService, resourceLinker, logFactory("CoreAuthenticationAPI")),
	}
}

// AuthenticateGitHub uses OAuth to authenticate the user using GitHub.
// This endpoint will redirect the browser to GitHub, which will then redirect back to
// the AuthenticateGitHubCallback handler.
func (a *CoreAuthenticationAPI) AuthenticateGitHub(w http.ResponseWriter, r *http.Request) {

	successRedirectURLStr := r.URL.Query().Get("success_url")
	errorRedirectURLStr := r.URL.Query().Get("error_url")

	if successRedirectURLStr == "" || errorRedirectURLStr == "" {
		a.Error(w, r, gerror.NewErrInternal().Wrap(errors.New("success_url or error_url not set on oauth initial call")))
		return
	}

	stateBytes := make([]byte, 16)

	_, err := rand.Read(stateBytes)
	if err != nil {
		a.Error(w, r, errors.Wrap(err, "error generating oauth state"))
		return
	}

	state := base64.URLEncoding.EncodeToString(stateBytes)

	pairs := map[string]string{
		sessionGitHubAuthStateKeyName:      state,
		sessionOAuthRedirectSuccessKeyName: successRedirectURLStr,
		sessionOAuthRedirectErrorKeyName:   errorRedirectURLStr,
	}

	err = a.setSessionValues(w, r, pairs)
	if err != nil {
		a.Error(w, r, err)
		return
	}

	url := a.config.GitHub.AuthCodeURL(state)

	http.Redirect(w, r, url, 302)
}

// AuthenticateGitHubCallback is the second step in the OAuth flow for GitHub authentication.
// It attempts to obtain a GitHub user auth token which is then matched up to a BuildBeaver
// user. On success a session cookie is issued and the browser is redirected to the success url,
// on error the browser is redirected to the error url.
func (a *CoreAuthenticationAPI) AuthenticateGitHubCallback(w http.ResponseWriter, r *http.Request) {

	session := a.getSession(r)
	expectedState := a.getSessionValue(session, sessionGitHubAuthStateKeyName)
	successRedirectURLStr := a.getSessionValue(session, sessionOAuthRedirectSuccessKeyName)
	errorRedirectURLStr := a.getSessionValue(session, sessionOAuthRedirectErrorKeyName)

	if expectedState == "" {
		a.Error(w, r, gerror.NewErrInternal().Wrap(errors.New("github_state not set on oauth callback")))
		return
	}
	if successRedirectURLStr == "" {
		a.Error(w, r, gerror.NewErrInternal().Wrap(errors.New("redirect_success_url not set on oauth callback")))
		return
	}
	if errorRedirectURLStr == "" {
		a.Error(w, r, gerror.NewErrInternal().Wrap(errors.New("redirect_error_url not set on oauth callback")))
		return
	}

	successRedirectURL, err := url.Parse(successRedirectURLStr)
	if err != nil {
		a.Error(w, r, gerror.NewErrValidationFailed("error parsing success redirect url").Wrap(err))
		return
	}

	errorRedirectURL, err := url.Parse(errorRedirectURLStr)
	if err != nil {
		a.Error(w, r, gerror.NewErrValidationFailed("error parsing error redirect url").Wrap(err))
		return
	}

	if r.URL.Query().Get("state") != expectedState {
		a.Errorf("Mismatched state")
		http.Redirect(w, r, errorRedirectURL.String(), 302)
		return
	}

	code := r.URL.Query().Get("code")

	oAuthToken, err := a.config.GitHub.Exchange(r.Context(), code)
	if err != nil {
		a.Errorf("Error exchanging code for token: %v", err)
		http.Redirect(w, r, errorRedirectURL.String(), 302)
		return
	}

	auth := &ghub.GitHubSCMAuthentication{
		Token: oAuthToken,
	}

	identity, err := a.authenticationService.AuthenticateSCMAuth(r.Context(), auth)

	if err != nil {
		a.Errorf("Error authenticating: %s", err)
		http.Redirect(w, r, errorRedirectURL.String(), 302)
		return
	}

	oAuthTokenJson, err := json.Marshal(oAuthToken)
	if err != nil {
		a.Errorf("Error serializing oauth token: %s", err)
		http.Redirect(w, r, errorRedirectURL.String(), 302)
		return
	}

	pairs := map[string]string{
		sessionIdentityIDKeyName: identity.ID.String(),
		sessionOAuthTokenKeyName: string(oAuthTokenJson),
	}

	err = a.setSessionValues(w, r, pairs)
	if err != nil {
		a.Errorf("Error issuing session: %s", err)
		http.Redirect(w, r, errorRedirectURL.String(), 302)
		return
	}

	a.Infof("Identity %s authenticated using GitHub", identity.ID.String())

	http.Redirect(w, r, successRedirectURL.String(), 302)
}

// SessionAuthenticator makes a middleware that authenticates requests using a session cookie
// from the request headers. If the request headers do not contain a session cookie then this is a no-op.
func (a *CoreAuthenticationAPI) SessionAuthenticator(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var (
			identityID models.IdentityID
			session    = a.getSession(r)
		)
		if session != nil {
			identityIDStr := a.getSessionValue(session, sessionIdentityIDKeyName)
			if identityIDStr != "" {
				id, err := models.ParseResourceID(identityIDStr)
				if err == nil {
					identityID = models.IdentityIDFromResourceID(id)
				}
			}

			oauthTokenStr := a.getSessionValue(session, sessionOAuthTokenKeyName)
			oauthToken := &oauth2.Token{}
			if oauthTokenStr != "" {
				err := json.Unmarshal([]byte(oauthTokenStr), oauthToken)
				if err != nil {
					a.Warnf("OAuth token could not be parsed from session data: %s", err)
					oauthToken = nil
				}
			}

			if identityID.Valid() {
				meta := &middleware.AuthenticationMeta{
					IdentityID:     identityID,
					CredentialType: models.CredentialTypeGitHubOAuth, // TODO stash/restore credential type in/from session
					OAuthToken:     oauthToken,
				}
				ctx := context.WithValue(r.Context(), AuthenticationMetaContextKeyName, meta)
				r = r.WithContext(ctx)
				a.Infof("Authenticated legal entity %s using session", identityID.String())
			}
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func (a *CoreAuthenticationAPI) getSession(r *http.Request) *sessions.Session {
	session, err := a.sessionStore.Get(r, sessionName)
	if err != nil {
		session, _ = a.sessionStore.New(r, sessionName)
	}
	return session
}

func (a *CoreAuthenticationAPI) setSessionValues(w http.ResponseWriter, r *http.Request, pairs map[string]string) error {
	session := a.getSession(r)
	for k, v := range pairs {
		session.Values[k] = v
	}
	err := session.Save(r, w)
	if err != nil {
		return errors.Wrap(err, "error saving session")
	}
	return nil
}

func (a *CoreAuthenticationAPI) setSessionValue(w http.ResponseWriter, r *http.Request, name string, value string) error {
	session := a.getSession(r)
	session.Values[name] = value
	err := session.Save(r, w)
	if err != nil {
		return errors.Wrap(err, "error saving session")
	}
	return nil
}

func (a *CoreAuthenticationAPI) getSessionValue(session *sessions.Session, name string) string {
	v, ok := session.Values[name]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
