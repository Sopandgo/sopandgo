package auth_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

const testPassword = "SuperSecret123!"

func codeAt(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	code, err := totp.GenerateCodeCustom(secret, at, totp.ValidateOpts{
		Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	return code
}

// newUser registers an account with a known password.
func newUser(t *testing.T, env *testenv.Env, email, role string) string {
	t.Helper()
	actor := "tf-actor"
	env.SeedTestUser(t, actor, "tf-actor@test.local", auth.RoleAdmin)
	id, err := env.AuthService.RegisterUser("Two Factor", email, role, &actor)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := env.AuthService.UpdateUserPassword(id, testPassword, &actor); err != nil {
		t.Fatalf("set password: %v", err)
	}
	return id
}

// enroll sets up and confirms TOTP and returns the secret and recovery codes.
func enroll(t *testing.T, env *testenv.Env, userID string) (string, []string) {
	t.Helper()
	setup, err := env.AuthService.BeginTOTPSetup(userID, testPassword)
	if err != nil {
		t.Fatalf("begin setup: %v", err)
	}
	if !strings.HasPrefix(setup.QRCodePNG, "data:image/png;base64,") || !strings.HasPrefix(setup.URI, "otpauth://totp/") {
		t.Fatalf("unexpected setup payload: %+v", setup.URI)
	}
	codes, err := env.AuthService.ConfirmTOTPSetup(userID, codeAt(t, setup.Secret, time.Now()), "")
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	return setup.Secret, codes
}

func countEvents(t *testing.T, env *testenv.Env, eventType, entityID string) int {
	t.Helper()
	var n int
	if err := env.Store.DB.QueryRow(
		`SELECT COUNT(*) FROM audit_events WHERE event_type = ? AND entity_id = ?`, eventType, entityID,
	).Scan(&n); err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	return n
}

func TestTwoFactor_EnrollAndVerifyTOTP(t *testing.T) {
	env := testenv.New(t)
	userID := newUser(t, env, "viewer@tf.local", auth.RoleViewer)

	if _, err := env.AuthService.BeginTOTPSetup(userID, "wrong-password"); !errors.Is(err, auth.ErrWrongCurrentPassword) {
		t.Fatalf("setup with wrong password: got %v", err)
	}

	setup, err := env.AuthService.BeginTOTPSetup(userID, testPassword)
	if err != nil {
		t.Fatalf("begin setup: %v", err)
	}
	user, _ := env.AuthService.GetUserByID(userID)
	if user.TwoFactorEnabled {
		t.Fatal("2FA must stay off until the first code is confirmed")
	}
	if _, err := env.AuthService.ConfirmTOTPSetup(userID, "000000", ""); !errors.Is(err, auth.ErrInvalidSecondFactor) {
		t.Fatalf("confirm with wrong code: got %v", err)
	}
	now := time.Now()
	codes, err := env.AuthService.ConfirmTOTPSetup(userID, codeAt(t, setup.Secret, now), "")
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if len(codes) != 0 {
		t.Fatalf("non-admins get no recovery codes, got %d", len(codes))
	}
	user, _ = env.AuthService.GetUserByID(userID)
	if !user.TwoFactorEnabled {
		t.Fatal("2FA should be on after confirm")
	}
	if countEvents(t, env, audit.EventTwoFactorEnabled, userID) != 1 {
		t.Fatal("expected one two_factor_enabled event")
	}

	// The confirm step's code is spent; the next step works once.
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorTOTP, codeAt(t, setup.Secret, now)); !errors.Is(err, auth.ErrInvalidSecondFactor) {
		t.Fatalf("replay of confirm code: got %v", err)
	}
	next := codeAt(t, setup.Secret, now.Add(30*time.Second))
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorTOTP, next); err != nil {
		t.Fatalf("next step code: %v", err)
	}
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorTOTP, next); !errors.Is(err, auth.ErrInvalidSecondFactor) {
		t.Fatalf("replayed code: got %v", err)
	}
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorTOTP, codeAt(t, setup.Secret, now.Add(5*time.Minute))); !errors.Is(err, auth.ErrInvalidSecondFactor) {
		t.Fatalf("code far outside the window: got %v", err)
	}
}

func TestTwoFactor_SetupNeedsEncryptionKey(t *testing.T) {
	env := testenv.New(t)
	userID := newUser(t, env, "nokey@tf.local", auth.RoleViewer)
	noKey := auth.NewService(env.Store.DB, env.AuditLogger, env.DataDir, nil)
	if _, err := noKey.BeginTOTPSetup(userID, testPassword); !errors.Is(err, auth.ErrNoEncryptionKey) {
		t.Fatalf("got %v", err)
	}
}

func TestTwoFactor_UnreadableSecretAfterKeyChange(t *testing.T) {
	env := testenv.New(t)
	userID := newUser(t, env, "rotated@tf.local", auth.RoleViewer)
	secret, _ := enroll(t, env, userID)

	otherKey := auth.NewService(env.Store.DB, env.AuditLogger, env.DataDir, []byte("abcdefghijabcdefghijabcdefghij12"))
	_, err := otherKey.VerifySecondFactor(userID, auth.SecondFactorTOTP, codeAt(t, secret, time.Now().Add(30*time.Second)))
	if !errors.Is(err, auth.ErrTwoFactorUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestTwoFactor_ReplaceAuthenticatorKeepsOldUntilConfirmed(t *testing.T) {
	env := testenv.New(t)
	userID := newUser(t, env, "replace@tf.local", auth.RoleViewer)
	oldSecret, _ := enroll(t, env, userID)

	setup, err := env.AuthService.BeginTOTPSetup(userID, testPassword)
	if err != nil {
		t.Fatalf("begin replacement: %v", err)
	}
	now := time.Now()
	// Old authenticator still signs in while the new one is pending.
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorTOTP, codeAt(t, oldSecret, now.Add(30*time.Second))); err != nil {
		t.Fatalf("old authenticator during replacement: %v", err)
	}
	if _, err := env.AuthService.ConfirmTOTPSetup(userID, codeAt(t, setup.Secret, now), ""); err != nil {
		t.Fatalf("confirm replacement: %v", err)
	}
	status, err := env.AuthService.TwoFactorStatus(userID)
	if err != nil || !status.Enabled || status.PendingSetup {
		t.Fatalf("status after replacement: %+v, %v", status, err)
	}
	// The old authenticator no longer works.
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorTOTP, codeAt(t, oldSecret, now.Add(30*time.Second))); !errors.Is(err, auth.ErrInvalidSecondFactor) {
		t.Fatalf("old authenticator after replacement: got %v", err)
	}
}

func TestTwoFactor_RecoveryCodesAdminOnly(t *testing.T) {
	env := testenv.New(t)
	adminID := newUser(t, env, "boss@tf.local", auth.RoleAdmin)
	_, codes := enroll(t, env, adminID)
	if len(codes) != 10 {
		t.Fatalf("admins get 10 recovery codes, got %d", len(codes))
	}

	// Lowercase and without the dash is accepted.
	sloppy := strings.ToLower(strings.ReplaceAll(codes[0], "-", ""))
	remaining, err := env.AuthService.VerifySecondFactor(adminID, auth.SecondFactorRecoveryCode, sloppy)
	if err != nil || remaining != 9 {
		t.Fatalf("use recovery code: remaining=%d err=%v", remaining, err)
	}
	if _, err := env.AuthService.VerifySecondFactor(adminID, auth.SecondFactorRecoveryCode, codes[0]); !errors.Is(err, auth.ErrInvalidSecondFactor) {
		t.Fatalf("a recovery code works once: got %v", err)
	}
	if countEvents(t, env, audit.EventTwoFactorRecoveryCodeUsed, adminID) != 1 {
		t.Fatal("expected one two_factor_recovery_code_used event")
	}

	// Demotion removes the codes; a non-admin cannot use one.
	actor := "tf-actor"
	if err := env.AuthService.UpdateUserRole(adminID, auth.RoleEditor, &actor); err != nil {
		t.Fatalf("demote: %v", err)
	}
	if _, err := env.AuthService.VerifySecondFactor(adminID, auth.SecondFactorRecoveryCode, codes[1]); !errors.Is(err, auth.ErrRecoveryCodesAdminOnly) {
		t.Fatalf("recovery code after demotion: got %v", err)
	}
	var left int
	env.Store.DB.QueryRow(`SELECT COUNT(*) FROM two_factor_recovery_codes WHERE user_id = ?`, adminID).Scan(&left)
	if left != 0 {
		t.Fatalf("demotion should delete recovery codes, %d left", left)
	}
	if _, err := env.AuthService.RegenerateRecoveryCodes(adminID, testPassword); !errors.Is(err, auth.ErrRecoveryCodesAdminOnly) {
		t.Fatalf("regenerate as editor: got %v", err)
	}
}

// setPolicy makes adminID an admin with an app (needed for any level above
// none) and sets the organization's policy with email switched on.
func setPolicy(t *testing.T, env *testenv.Env, adminID, policy string) {
	t.Helper()
	if user, _ := env.AuthService.GetUserByID(adminID); !user.TwoFactorEnabled {
		enroll(t, env, adminID)
	}
	if err := env.AuthService.SetTwoFactorPolicy(policy, adminID, true); err != nil {
		t.Fatalf("set policy %s: %v", policy, err)
	}
}

func TestTwoFactor_EmailCode(t *testing.T) {
	env := testenv.New(t)
	adminID := newUser(t, env, "mail-admin@tf.local", auth.RoleAdmin)
	userID := newUser(t, env, "mail@tf.local", auth.RoleViewer)

	// Policy none: email codes are not accepted.
	if _, err := env.AuthService.IssueEmailCode(userID); !errors.Is(err, auth.ErrEmailCodesNotAllowed) {
		t.Fatalf("email code under none: got %v", err)
	}
	setPolicy(t, env, adminID, auth.TwoFactorPolicyEmail)

	// No app needed: the invite already proved the mailbox.
	code, err := env.AuthService.IssueEmailCode(userID)
	if err != nil || len(code) != 6 {
		t.Fatalf("issue: %q %v", code, err)
	}
	if _, err := env.AuthService.IssueEmailCode(userID); !errors.Is(err, auth.ErrEmailCodeThrottled) {
		t.Fatalf("second send within a minute: got %v", err)
	}
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorEmail, code); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorEmail, code); !errors.Is(err, auth.ErrInvalidSecondFactor) {
		t.Fatalf("email code works once: got %v", err)
	}

	// Five wrong guesses burn the code.
	env.Store.DB.Exec(`DELETE FROM two_factor_email_codes`)
	code, err = env.AuthService.IssueEmailCode(userID)
	if err != nil {
		t.Fatalf("issue again: %v", err)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < 5; i++ {
		env.AuthService.VerifySecondFactor(userID, auth.SecondFactorEmail, wrong)
	}
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorEmail, code); !errors.Is(err, auth.ErrInvalidSecondFactor) {
		t.Fatalf("code after 5 wrong attempts: got %v", err)
	}

	// Expired codes fail.
	env.Store.DB.Exec(`DELETE FROM two_factor_email_codes`)
	code, _ = env.AuthService.IssueEmailCode(userID)
	env.Store.DB.Exec(`UPDATE two_factor_email_codes SET expires_at = ?`, time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano))
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorEmail, code); !errors.Is(err, auth.ErrInvalidSecondFactor) {
		t.Fatalf("expired code: got %v", err)
	}

	// Admins never get email codes.
	if _, err := env.AuthService.IssueEmailCode(adminID); !errors.Is(err, auth.ErrEmailCodesNotAllowed) {
		t.Fatalf("email code for an admin: got %v", err)
	}

	// Raising the policy to app-only stops email codes, even one already sent.
	env.Store.DB.Exec(`DELETE FROM two_factor_email_codes`)
	code, _ = env.AuthService.IssueEmailCode(userID)
	setPolicy(t, env, adminID, auth.TwoFactorPolicyTOTP)
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorEmail, code); !errors.Is(err, auth.ErrEmailCodesNotAllowed) {
		t.Fatalf("email code after switching to app-only: got %v", err)
	}
}

func TestTwoFactor_PolicyRules(t *testing.T) {
	env := testenv.New(t)
	adminID := newUser(t, env, "policy-admin@tf.local", auth.RoleAdmin)
	otherAdmin := newUser(t, env, "policy-other@tf.local", auth.RoleAdmin)
	userID := newUser(t, env, "policy-user@tf.local", auth.RoleViewer)

	methods := func(id string, mailOn bool) string {
		t.Helper()
		u, err := env.AuthService.GetUserByID(id)
		if err != nil {
			t.Fatal(err)
		}
		m, err := env.AuthService.SecondFactorMethods(u, mailOn)
		if err != nil {
			t.Fatal(err)
		}
		return fmt.Sprint(auth.SecondFactorRequired(u), m)
	}

	// none: password only, and nobody must enroll.
	if got := methods(userID, true); got != "false []" {
		t.Fatalf("none, viewer: %s", got)
	}

	// Raising the level needs your own app first.
	if err := env.AuthService.SetTwoFactorPolicy(auth.TwoFactorPolicyEmail, adminID, true); !errors.Is(err, auth.ErrActorNeedsTwoFactor) {
		t.Fatalf("raise without own app: got %v", err)
	}
	enroll(t, env, adminID)
	if err := env.AuthService.SetTwoFactorPolicy(auth.TwoFactorPolicyEmail, adminID, false); !errors.Is(err, auth.ErrEmailTransportOff) {
		t.Fatalf("email level with mail off: got %v", err)
	}
	if err := env.AuthService.SetTwoFactorPolicy("sms", adminID, true); !errors.Is(err, auth.ErrInvalidTwoFactorPolicy) {
		t.Fatalf("unknown policy: got %v", err)
	}
	noKey := auth.NewService(env.Store.DB, env.AuditLogger, env.DataDir, nil)
	if err := noKey.SetTwoFactorPolicy(auth.TwoFactorPolicyTOTP, adminID, true); !errors.Is(err, auth.ErrNoEncryptionKey) {
		t.Fatalf("raise without key: got %v", err)
	}

	// email: normal users get emailed codes without an app; admins must enroll.
	setPolicy(t, env, adminID, auth.TwoFactorPolicyEmail)
	if n := countEvents(t, env, audit.EventTwoFactorPolicyUpdated, audit.EntitySystem); n != 1 {
		t.Fatalf("expected one two_factor_policy_updated event, got %d", n)
	}
	if got := methods(userID, true); got != "true [email]" {
		t.Fatalf("email, viewer: %s", got)
	}
	// Mail switched off later: a second factor is still required, with no way to pass it.
	if got := methods(userID, false); got != "true []" {
		t.Fatalf("email, viewer, mail off: %s", got)
	}
	viewer, _ := env.AuthService.GetUserByID(userID)
	if viewer.MustEnrollTwoFactor {
		t.Fatal("email policy must not force an app on normal users")
	}
	other, _ := env.AuthService.GetUserByID(otherAdmin)
	if !other.MustEnrollTwoFactor || auth.SecondFactorRequired(other) {
		t.Fatalf("admin without app under email policy: must enroll, no email codes: %+v", other)
	}

	// A normal user may add an app and drop it again under email; admins may not.
	enroll(t, env, userID)
	if got := methods(userID, true); got != "true [totp email]" {
		t.Fatalf("email, viewer with app: %s", got)
	}
	if err := env.AuthService.DisableTwoFactor(userID, testPassword); err != nil {
		t.Fatalf("viewer removes app under email policy: %v", err)
	}
	if err := env.AuthService.DisableTwoFactor(adminID, testPassword); !errors.Is(err, auth.ErrTwoFactorRequired) {
		t.Fatalf("admin removes app under email policy: got %v", err)
	}

	// totp: everyone needs an app; email codes are gone.
	setPolicy(t, env, adminID, auth.TwoFactorPolicyTOTP)
	viewer, _ = env.AuthService.GetUserByID(userID)
	if !viewer.MustEnrollTwoFactor {
		t.Fatal("app-only policy must require an app")
	}
	if got := methods(userID, true); got != "false []" {
		t.Fatalf("totp, viewer without app (signs in, then must enroll): %s", got)
	}
	status, err := env.AuthService.TwoFactorStatus(userID)
	if err != nil || status.Policy != auth.TwoFactorPolicyTOTP || !status.Required || status.EmailCodes {
		t.Fatalf("status under totp: %+v %v", status, err)
	}

	// Lowering to none is always allowed, and admins may then remove their app.
	if err := env.AuthService.SetTwoFactorPolicy(auth.TwoFactorPolicyNone, otherAdmin, false); err != nil {
		t.Fatalf("lower to none: %v", err)
	}
	if err := env.AuthService.DisableTwoFactor(adminID, "nope"); !errors.Is(err, auth.ErrWrongCurrentPassword) {
		t.Fatalf("disable with wrong password: got %v", err)
	}
	if err := env.AuthService.DisableTwoFactor(adminID, testPassword); err != nil {
		t.Fatalf("disable: %v", err)
	}
	var codes int
	env.Store.DB.QueryRow(`SELECT COUNT(*) FROM two_factor_recovery_codes WHERE user_id = ?`, adminID).Scan(&codes)
	if codes != 0 {
		t.Fatalf("disable should remove recovery codes, %d left", codes)
	}
}

func TestTwoFactor_AdminResetRefusesAdmins(t *testing.T) {
	env := testenv.New(t)
	actor := newUser(t, env, "resetter@tf.local", auth.RoleAdmin)
	otherAdmin := newUser(t, env, "other-admin@tf.local", auth.RoleAdmin)
	userID := newUser(t, env, "reset-me@tf.local", auth.RoleEditor)
	enroll(t, env, otherAdmin)
	enroll(t, env, userID)
	session, _ := env.AuthService.CreateSession(userID)

	if err := env.AuthService.ResetTwoFactor(otherAdmin, &actor); !errors.Is(err, auth.ErrCannotResetAdmin) {
		t.Fatalf("reset admin: got %v", err)
	}
	if err := env.AuthService.ResetTwoFactor(userID, &actor); err != nil {
		t.Fatalf("reset user: %v", err)
	}
	user, _ := env.AuthService.GetUserByID(userID)
	if user.TwoFactorEnabled {
		t.Fatal("2FA should be cleared")
	}
	if _, _, err := env.AuthService.ValidateSession(session); err == nil {
		t.Fatal("reset should sign the user out")
	}
	if countEvents(t, env, audit.EventTwoFactorReset, userID) != 1 {
		t.Fatal("expected one two_factor_reset event")
	}
}

func TestBreakGlass_ResetAdmin(t *testing.T) {
	env := testenv.New(t)
	adminID := newUser(t, env, "locked@tf.local", auth.RoleAdmin)
	enroll(t, env, adminID)
	session, _ := env.AuthService.CreateSession(adminID)
	env.Store.DB.Exec(`UPDATE users SET is_active = 0 WHERE id = ?`, adminID)

	bg := auth.BreakGlassContext{Hostname: "lab-host", OSUser: "root"}
	res, err := env.AuthService.BreakGlassResetAdmin("  LOCKED@tf.local ", bg)
	if err != nil {
		t.Fatalf("break-glass: %v", err)
	}
	if !res.Reactivated || res.SessionsRevoked != 1 || len(res.Cleared) != 2 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if auth.ValidatePassword(res.TemporaryPassword) != nil {
		t.Fatal("temporary password must satisfy the password rules")
	}

	user, hash, _ := env.AuthService.GetUserByEmail("locked@tf.local")
	if user.TwoFactorEnabled || !user.IsActive || !user.MustChangePassword {
		t.Fatalf("account after break-glass: %+v", user)
	}
	if auth.VerifyPassword(res.TemporaryPassword, hash) != nil {
		t.Fatal("temporary password should sign in")
	}
	if _, _, err := env.AuthService.ValidateSession(session); err == nil {
		t.Fatal("break-glass should sign out every session")
	}

	var payload string
	var actor *string
	if err := env.Store.DB.QueryRow(`
		SELECT payload, actor_user_id FROM audit_events WHERE event_type = ? AND entity_id = ?
	`, audit.EventAdminBreakGlassReset, adminID).Scan(&payload, &actor); err != nil {
		t.Fatalf("break-glass audit event: %v", err)
	}
	if actor != nil {
		t.Fatalf("break-glass actor should be empty, got %v", *actor)
	}
	for _, want := range []string{`"via":"cli"`, `"hostname":"lab-host"`, `"os_user":"root"`, `"reactivated":true`} {
		if !strings.Contains(payload, want) {
			t.Fatalf("payload %s is missing %s", payload, want)
		}
	}
	if strings.Contains(payload, res.TemporaryPassword) {
		t.Fatal("the temporary password must never be logged")
	}
	if n := countEvents(t, env, audit.EventAdminBreakGlassReset, adminID); n != 1 {
		t.Fatalf("expected one admin_break_glass_reset event, got %d", n)
	}
	if report, err := env.AuditLogger.VerifyEntireChain(); err != nil || report.Invalid != 0 {
		t.Fatalf("audit chain after break-glass: %+v %v", report, err)
	}
}

func TestBreakGlass_RefusedRunsAreAudited(t *testing.T) {
	env := testenv.New(t)
	editorID := newUser(t, env, "editor@tf.local", auth.RoleEditor)
	bg := auth.BreakGlassContext{Hostname: "lab-host"}

	if _, err := env.AuthService.BreakGlassResetAdmin("ghost@tf.local", bg); !errors.Is(err, auth.ErrBreakGlassUserNotFound) {
		t.Fatalf("unknown email: got %v", err)
	}
	if countEvents(t, env, audit.EventAdminBreakGlassFailed, audit.EntitySystem) != 1 {
		t.Fatal("unknown email should write admin_break_glass_failed")
	}

	_, hashBefore, _ := env.AuthService.GetUserByEmail("editor@tf.local")
	if _, err := env.AuthService.BreakGlassResetAdmin("editor@tf.local", bg); !errors.Is(err, auth.ErrBreakGlassNotAdmin) {
		t.Fatalf("non-admin: got %v", err)
	}
	_, hashAfter, _ := env.AuthService.GetUserByEmail("editor@tf.local")
	if hashBefore != hashAfter {
		t.Fatal("a refused run must not change the password")
	}
	if countEvents(t, env, audit.EventAdminBreakGlassFailed, editorID) != 1 {
		t.Fatal("non-admin target should write admin_break_glass_failed")
	}

	env.AuthService.LogBreakGlassAborted("editor@tf.local", bg)
	if countEvents(t, env, audit.EventAdminBreakGlassFailed, editorID) != 2 {
		t.Fatal("an aborted run should write admin_break_glass_failed")
	}
}

func TestBreakGlass_AuditFailureRollsBack(t *testing.T) {
	env := testenv.New(t)
	adminID := newUser(t, env, "rollback@tf.local", auth.RoleAdmin)
	enroll(t, env, adminID)
	_, hashBefore, _ := env.AuthService.GetUserByEmail("rollback@tf.local")

	// Break the audit table so the in-transaction write fails.
	if _, err := env.Store.DB.Exec(`ALTER TABLE audit_events RENAME TO audit_events_hidden`); err != nil {
		t.Fatalf("hide audit table: %v", err)
	}
	_, err := env.AuthService.BreakGlassResetAdmin("rollback@tf.local", auth.BreakGlassContext{})
	if _, rerr := env.Store.DB.Exec(`ALTER TABLE audit_events_hidden RENAME TO audit_events`); rerr != nil {
		t.Fatalf("restore audit table: %v", rerr)
	}
	if err == nil {
		t.Fatal("break-glass must fail when the audit write fails")
	}

	user, hashAfter, _ := env.AuthService.GetUserByEmail("rollback@tf.local")
	if hashAfter != hashBefore || !user.TwoFactorEnabled || user.MustChangePassword {
		t.Fatalf("nothing may change without an audit entry: %+v", user)
	}
}

func TestTwoFactor_EmailCooldownSurvivesUseAndBurn(t *testing.T) {
	env := testenv.New(t)
	adminID := newUser(t, env, "cool-admin@tf.local", auth.RoleAdmin)
	userID := newUser(t, env, "cool@tf.local", auth.RoleViewer)
	setPolicy(t, env, adminID, auth.TwoFactorPolicyEmail)

	// Moves the last send two minutes into the past, as if the cooldown ran out.
	rewind := func() {
		t.Helper()
		if _, err := env.Store.DB.Exec(`UPDATE two_factor_email_codes SET created_at = ?`,
			time.Now().Add(-2*time.Minute).UTC().Format(time.RFC3339Nano)); err != nil {
			t.Fatal(err)
		}
	}

	// A code used successfully still counts toward the cooldown.
	code, err := env.AuthService.IssueEmailCode(userID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorEmail, code); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if _, err := env.AuthService.IssueEmailCode(userID); !errors.Is(err, auth.ErrEmailCodeThrottled) {
		t.Fatalf("resend right after a successful use: got %v", err)
	}

	// A code burned by five wrong guesses still counts toward the cooldown.
	rewind()
	code, err = env.AuthService.IssueEmailCode(userID)
	if err != nil {
		t.Fatalf("send after the cooldown: %v", err)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < 5; i++ {
		env.AuthService.VerifySecondFactor(userID, auth.SecondFactorEmail, wrong)
	}
	if _, err := env.AuthService.IssueEmailCode(userID); !errors.Is(err, auth.ErrEmailCodeThrottled) {
		t.Fatalf("resend right after burning a code: got %v", err)
	}
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorEmail, code); !errors.Is(err, auth.ErrInvalidSecondFactor) {
		t.Fatalf("burned code: got %v", err)
	}

	// Once a minute has passed, a new code is sent and works.
	rewind()
	code, err = env.AuthService.IssueEmailCode(userID)
	if err != nil {
		t.Fatalf("send after the cooldown: %v", err)
	}
	if _, err := env.AuthService.VerifySecondFactor(userID, auth.SecondFactorEmail, code); err != nil {
		t.Fatalf("fresh code: %v", err)
	}
}
