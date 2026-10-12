package api_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

type tfClient struct {
	t      *testing.T
	env    *testenv.Env
	nextIP int
}

// do sends a request from a fresh client address, so the per-IP login limiter
// does not interfere with tests that are not about it.
func (c *tfClient) do(method, path, token string, body any) *httptest.ResponseRecorder {
	c.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	c.nextIP++
	req.RemoteAddr = fmt.Sprintf("198.51.100.%d:4000", c.nextIP%250+1)
	rec := httptest.NewRecorder()
	c.env.API.ServeHTTP(rec, req)
	return rec
}

func tfCode(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	code, err := totp.GenerateCodeCustom(secret, at, totp.ValidateOpts{Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func tfUser(t *testing.T, env *testenv.Env, email, role string) string {
	t.Helper()
	actor := "tf-api-actor"
	env.SeedTestUser(t, actor, "tf-api-actor@test.local", auth.RoleAdmin)
	id, err := env.AuthService.RegisterUser("TF", email, role, &actor)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.AuthService.UpdateUserPassword(id, "SuperSecret123!", &actor); err != nil {
		t.Fatal(err)
	}
	return id
}

func tfEnroll(t *testing.T, env *testenv.Env, userID string) string {
	t.Helper()
	setup, err := env.AuthService.BeginTOTPSetup(userID, "SuperSecret123!")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.AuthService.ConfirmTOTPSetup(userID, tfCode(t, setup.Secret, time.Now()), ""); err != nil {
		t.Fatal(err)
	}
	return setup.Secret
}

func TestAPI_TwoFactorLogin(t *testing.T) {
	env := testenv.New(t)
	c := &tfClient{t: t, env: env}
	userID := tfUser(t, env, "login@tf.local", auth.RoleEditor)
	secret := tfEnroll(t, env, userID)

	rec := c.do(http.MethodPost, "/api/auth/login", "", map[string]string{"email": "login@tf.local", "password": "SuperSecret123!"})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	var challenge struct {
		Required bool     `json:"two_factor_required"`
		Token    string   `json:"challenge_token"`
		Methods  []string `json:"methods"`
		Refresh  string   `json:"refresh_token"`
	}
	json.NewDecoder(rec.Body).Decode(&challenge)
	if !challenge.Required || challenge.Token == "" || challenge.Refresh != "" {
		t.Fatalf("password alone must not sign in: %+v", challenge)
	}
	// Policy none: email codes are not offered, even with mail switched on.
	if fmt.Sprint(challenge.Methods) != "[totp]" {
		t.Fatalf("methods: %v", challenge.Methods)
	}
	rec = c.do(http.MethodPost, "/api/auth/login/two-factor", "", map[string]string{
		"challenge_token": challenge.Token, "method": "email", "code": "123456",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("method the policy does not allow: %d", rec.Code)
	}

	// The challenge is not an access token.
	if rec := c.do(http.MethodGet, "/api/auth/me", challenge.Token, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("challenge token used as bearer: %d", rec.Code)
	}

	rec = c.do(http.MethodPost, "/api/auth/login/two-factor", "", map[string]string{
		"challenge_token": challenge.Token, "method": "totp", "code": "000000",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code: %d", rec.Code)
	}

	rec = c.do(http.MethodPost, "/api/auth/login/two-factor", "", map[string]string{
		"challenge_token": challenge.Token, "method": "totp", "code": tfCode(t, secret, time.Now().Add(30*time.Second)),
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("second factor: %d %s", rec.Code, rec.Body)
	}
	var tokens map[string]string
	json.NewDecoder(rec.Body).Decode(&tokens)
	if tokens["access_token"] == "" || tokens["refresh_token"] == "" {
		t.Fatalf("expected tokens, got %v", tokens)
	}

	rec = c.do(http.MethodPost, "/api/auth/login/two-factor", "", map[string]string{
		"challenge_token": "v4.local.garbage", "code": "123456",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad challenge: %d", rec.Code)
	}
}

func TestAPI_TwoFactorEmailPolicyLogin(t *testing.T) {
	env := testenv.New(t)
	c := &tfClient{t: t, env: env}
	adminID := tfUser(t, env, "email-admin@tf.local", auth.RoleAdmin)
	tfUser(t, env, "email-viewer@tf.local", auth.RoleViewer)
	tfEnroll(t, env, adminID)
	if err := env.AuthService.SetTwoFactorPolicy(auth.TwoFactorPolicyEmail, adminID, true); err != nil {
		t.Fatal(err)
	}

	// A viewer without an app gets a code by email right after the password.
	before := env.MockMailSender.SentCount
	rec := c.do(http.MethodPost, "/api/auth/login", "", map[string]string{"email": "email-viewer@tf.local", "password": "SuperSecret123!"})
	var challenge struct {
		Token     string   `json:"challenge_token"`
		Methods   []string `json:"methods"`
		EmailSent bool     `json:"email_sent"`
	}
	json.NewDecoder(rec.Body).Decode(&challenge)
	if rec.Code != http.StatusOK || fmt.Sprint(challenge.Methods) != "[email]" || !challenge.EmailSent {
		t.Fatalf("login under email policy: %d %+v", rec.Code, challenge)
	}
	if env.MockMailSender.SentCount != before+1 {
		t.Fatalf("expected one sign-in email, got %d", env.MockMailSender.SentCount-before)
	}
	rec = c.do(http.MethodPost, "/api/auth/login/two-factor/email", "", map[string]string{"challenge_token": challenge.Token})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("resend within a minute: %d", rec.Code)
	}

	// Swap in a known code (the mock sender does not keep the body).
	sum := sha256.Sum256([]byte("123456"))
	env.Store.DB.Exec(`UPDATE two_factor_email_codes SET code_hash = ?`, hex.EncodeToString(sum[:]))
	rec = c.do(http.MethodPost, "/api/auth/login/two-factor", "", map[string]string{
		"challenge_token": challenge.Token, "method": "email", "code": "123456",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("emailed code: %d %s", rec.Code, rec.Body)
	}

	// The admin is not offered email codes.
	rec = c.do(http.MethodPost, "/api/auth/login", "", map[string]string{"email": "email-admin@tf.local", "password": "SuperSecret123!"})
	json.NewDecoder(rec.Body).Decode(&challenge)
	if fmt.Sprint(challenge.Methods) != "[totp recovery_code]" {
		t.Fatalf("admin methods: %v", challenge.Methods)
	}

	// Email switched off later: the password alone must not sign in.
	if err := env.SMTPSettings.SetMailMode("manual_links"); err != nil {
		t.Fatal(err)
	}
	rec = c.do(http.MethodPost, "/api/auth/login", "", map[string]string{"email": "email-viewer@tf.local", "password": "SuperSecret123!"})
	if rec.Code != http.StatusServiceUnavailable || bytes.Contains(rec.Body.Bytes(), []byte("refresh_token")) {
		t.Fatalf("login with mail off: %d %s", rec.Code, rec.Body)
	}
}

func TestAPI_TwoFactorPerUserLimiter(t *testing.T) {
	env := testenv.New(t)
	c := &tfClient{t: t, env: env}
	userID := tfUser(t, env, "limited@tf.local", auth.RoleViewer)
	tfEnroll(t, env, userID)
	challenge, _ := auth.GenerateTwoFactorChallenge(userID)

	// Each guess comes from a different address; the per-user limit still applies.
	var last int
	for i := 0; i < 6; i++ {
		last = c.do(http.MethodPost, "/api/auth/login/two-factor", "", map[string]string{
			"challenge_token": challenge, "code": "000000",
		}).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("sixth guess: got %d, want 429", last)
	}
}

func TestAPI_TwoFactorEnrollmentGuard(t *testing.T) {
	env := testenv.New(t)
	c := &tfClient{t: t, env: env}
	adminID := tfUser(t, env, "guard-admin@tf.local", auth.RoleAdmin)
	viewerID := tfUser(t, env, "guard-viewer@tf.local", auth.RoleViewer)
	tfEnroll(t, env, adminID)
	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

	rec := c.do(http.MethodPatch, "/api/admin/settings/security", adminToken, map[string]string{"two_factor_policy": "totp"})
	if rec.Code != http.StatusOK {
		t.Fatalf("require an app: %d %s", rec.Code, rec.Body)
	}

	if rec := c.do(http.MethodGet, "/api/sops", viewerToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("non-enrolled user reached SOPs: %d", rec.Code)
	}
	rec = c.do(http.MethodGet, "/api/auth/me", viewerToken, nil)
	var me auth.User
	json.NewDecoder(rec.Body).Decode(&me)
	if rec.Code != http.StatusOK || !me.MustEnrollTwoFactor {
		t.Fatalf("me: %d %+v", rec.Code, me)
	}
	rec = c.do(http.MethodPost, "/api/auth/me/two-factor/setup", viewerToken, map[string]string{"current_password": "SuperSecret123!"})
	if rec.Code != http.StatusOK {
		t.Fatalf("setup while enrollment is required: %d %s", rec.Code, rec.Body)
	}
	var setup auth.TOTPSetup
	json.NewDecoder(rec.Body).Decode(&setup)
	rec = c.do(http.MethodPost, "/api/auth/me/two-factor/confirm", viewerToken, map[string]string{"code": tfCode(t, setup.Secret, time.Now())})
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body)
	}
	if rec := c.do(http.MethodGet, "/api/sops", viewerToken, nil); rec.Code != http.StatusOK {
		t.Fatalf("enrolled user should reach SOPs: %d", rec.Code)
	}
	rec = c.do(http.MethodDelete, "/api/auth/me/two-factor", viewerToken, map[string]string{"current_password": "SuperSecret123!"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("disable while required: %d", rec.Code)
	}
}

func TestAPI_AdminResetTwoFactor(t *testing.T) {
	env := testenv.New(t)
	c := &tfClient{t: t, env: env}
	adminID := tfUser(t, env, "reset-admin@tf.local", auth.RoleAdmin)
	otherAdmin := tfUser(t, env, "reset-other@tf.local", auth.RoleAdmin)
	viewerID := tfUser(t, env, "reset-viewer@tf.local", auth.RoleViewer)
	tfEnroll(t, env, otherAdmin)
	tfEnroll(t, env, viewerID)
	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

	if rec := c.do(http.MethodDelete, "/api/admin/users/"+viewerID+"/two-factor", viewerToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("viewer resetting 2FA: %d", rec.Code)
	}
	if rec := c.do(http.MethodDelete, "/api/admin/users/"+otherAdmin+"/two-factor", adminToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("resetting an admin: %d", rec.Code)
	}
	if rec := c.do(http.MethodDelete, "/api/admin/users/"+viewerID+"/two-factor", adminToken, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("resetting a viewer: %d %s", rec.Code, rec.Body)
	}
	user, _ := env.AuthService.GetUserByID(viewerID)
	if user.TwoFactorEnabled {
		t.Fatal("2FA should be cleared")
	}

	rec := c.do(http.MethodGet, "/api/admin/users", adminToken, nil)
	var users []auth.User
	json.NewDecoder(rec.Body).Decode(&users)
	for _, u := range users {
		if u.ID == otherAdmin && !u.TwoFactorEnabled {
			t.Fatal("admin user list should show 2FA status")
		}
	}
}

func TestAPI_SecuritySettingsNeedsOwnTwoFactor(t *testing.T) {
	env := testenv.New(t)
	c := &tfClient{t: t, env: env}
	adminID := tfUser(t, env, "settings-admin@tf.local", auth.RoleAdmin)
	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)

	rec := c.do(http.MethodPatch, "/api/admin/settings/security", adminToken, map[string]string{"two_factor_policy": "email"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("raise without own app: %d", rec.Code)
	}
	rec = c.do(http.MethodPatch, "/api/admin/settings/security", adminToken, map[string]string{"two_factor_policy": "sms"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown policy: %d", rec.Code)
	}
	rec = c.do(http.MethodGet, "/api/admin/settings/security", adminToken, nil)
	var out map[string]any
	json.NewDecoder(rec.Body).Decode(&out)
	if out["two_factor_policy"] != "none" || out["encryption_key_configured"] != true || out["email_transport_on"] != true {
		t.Fatalf("settings: %v", out)
	}
}

// tfAdminWithCodes enrolls an admin and returns the recovery codes.
func tfAdminWithCodes(t *testing.T, env *testenv.Env, email string) (string, []string) {
	t.Helper()
	adminID := tfUser(t, env, email, auth.RoleAdmin)
	setup, err := env.AuthService.BeginTOTPSetup(adminID, "SuperSecret123!")
	if err != nil {
		t.Fatal(err)
	}
	codes, err := env.AuthService.ConfirmTOTPSetup(adminID, tfCode(t, setup.Secret, time.Now()), "")
	if err != nil || len(codes) < 3 {
		t.Fatalf("confirm: %v (%d codes)", err, len(codes))
	}
	return adminID, codes
}

func TestAPI_TwoFactorChallengeIsSingleUse(t *testing.T) {
	env := testenv.New(t)
	c := &tfClient{t: t, env: env}
	adminID, codes := tfAdminWithCodes(t, env, "single@tf.local")
	challenge, _ := auth.GenerateTwoFactorChallenge(adminID)

	verify := func(code string) int {
		return c.do(http.MethodPost, "/api/auth/login/two-factor", "", map[string]string{
			"challenge_token": challenge, "method": "recovery_code", "code": code,
		}).Code
	}
	if got := verify(codes[0]); got != http.StatusOK {
		t.Fatalf("first sign-in: %d", got)
	}
	// Same challenge, another valid code: no second session.
	if got := verify(codes[1]); got != http.StatusUnauthorized {
		t.Fatalf("reused challenge: %d", got)
	}
	// The rejected attempt did not spend codes[1]; it works with a fresh challenge.
	challenge, _ = auth.GenerateTwoFactorChallenge(adminID)
	if got := verify(codes[1]); got != http.StatusOK {
		t.Fatalf("code after a rejected reuse: %d", got)
	}
	if got := c.do(http.MethodPost, "/api/auth/login/two-factor/email", "", map[string]string{"challenge_token": challenge}).Code; got != http.StatusUnauthorized {
		t.Fatalf("email send with a used challenge: %d", got)
	}
}

func TestAPI_TwoFactorChallengeRaceOpensOneSession(t *testing.T) {
	env := testenv.New(t)
	adminID, codes := tfAdminWithCodes(t, env, "race@tf.local")
	challenge, _ := auth.GenerateTwoFactorChallenge(adminID)

	// Two requests with the same challenge and different valid codes at once.
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for i, code := range codes[:2] {
		wg.Add(1)
		go func(i int, code string) {
			defer wg.Done()
			b, _ := json.Marshal(map[string]string{"challenge_token": challenge, "method": "recovery_code", "code": code})
			req, _ := http.NewRequest(http.MethodPost, "/api/auth/login/two-factor", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = fmt.Sprintf("203.0.113.%d:4000", i+1)
			rec := httptest.NewRecorder()
			env.API.ServeHTTP(rec, req)
			results <- rec.Code
		}(i, code)
	}
	wg.Wait()
	close(results)
	ok := 0
	for code := range results {
		if code == http.StatusOK {
			ok++
		}
	}
	if ok != 1 {
		t.Fatalf("expected exactly one session, got %d", ok)
	}
}

func TestAPI_EmailCodeCooldownAfterUse(t *testing.T) {
	env := testenv.New(t)
	c := &tfClient{t: t, env: env}
	adminID := tfUser(t, env, "cool-api-admin@tf.local", auth.RoleAdmin)
	tfUser(t, env, "cool-api@tf.local", auth.RoleViewer)
	tfEnroll(t, env, adminID)
	if err := env.AuthService.SetTwoFactorPolicy(auth.TwoFactorPolicyEmail, adminID, true); err != nil {
		t.Fatal(err)
	}

	login := func() (string, bool) {
		rec := c.do(http.MethodPost, "/api/auth/login", "", map[string]string{"email": "cool-api@tf.local", "password": "SuperSecret123!"})
		var ch struct {
			Token     string `json:"challenge_token"`
			EmailSent bool   `json:"email_sent"`
		}
		json.NewDecoder(rec.Body).Decode(&ch)
		return ch.Token, ch.EmailSent
	}
	token, sent := login()
	if !sent {
		t.Fatal("first sign-in should mail a code")
	}
	sum := sha256.Sum256([]byte("123456"))
	env.Store.DB.Exec(`UPDATE two_factor_email_codes SET code_hash = ?`, hex.EncodeToString(sum[:]))
	if got := c.do(http.MethodPost, "/api/auth/login/two-factor", "", map[string]string{
		"challenge_token": token, "method": "email", "code": "123456",
	}).Code; got != http.StatusOK {
		t.Fatalf("sign in with the emailed code: %d", got)
	}

	// Signing in again right away does not mail another code.
	before := env.MockMailSender.SentCount
	if _, sent := login(); sent || env.MockMailSender.SentCount != before {
		t.Fatalf("a second code was mailed within the cooldown (sent=%v)", sent)
	}
}
