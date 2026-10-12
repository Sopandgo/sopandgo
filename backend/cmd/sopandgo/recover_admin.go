package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/notify"
	"github.com/sopandgo/sopandgo/backend/internal/secrets"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

// runCommand runs a host-only maintenance command and returns the exit code.
// These commands have no HTTP route: only someone with a shell in the
// container (Coolify Terminal, docker exec) can run them.
func runCommand(name string, args []string) int {
	switch name {
	case "recover-admin":
		return recoverAdmin(args, os.Stdin, os.Stdout, os.Stderr)
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", name)
		printUsage(os.Stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  sopandgo                                  start the server
  sopandgo recover-admin --list             list admin accounts
  sopandgo recover-admin --email <email>    reset a locked-out admin's sign-in

recover-admin removes the admin's two-factor authentication and recovery codes,
sets a temporary password that must be changed at the next sign-in, reactivates
the account, signs out every session, and writes admin_break_glass_reset to the
audit log. Run it inside the sopandgo container; DATA_DIR must point at the
live data directory.
`)
}

func recoverAdmin(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("recover-admin", flag.ContinueOnError)
	fs.SetOutput(stderr)
	list := fs.Bool("list", false, "list admin accounts and exit")
	email := fs.String("email", "", "email of the admin account to reset")
	yes := fs.Bool("yes", false, "skip the confirmation prompt")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !*list && strings.TrimSpace(*email) == "" {
		fmt.Fprintln(stderr, "recover-admin: pass --list or --email <email>")
		return 2
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	// Never create a fresh database: a wrong DATA_DIR would otherwise
	// "succeed" against an empty instance.
	if _, err := os.Stat(filepath.Join(dataDir, "app.db")); err != nil {
		fmt.Fprintf(stderr, "recover-admin: no database at %s (set DATA_DIR to the live data directory): %v\n",
			filepath.Join(dataDir, "app.db"), err)
		return 1
	}

	store, err := storage.Open(dataDir)
	if err != nil {
		fmt.Fprintf(stderr, "recover-admin: cannot open database: %v\n", err)
		return 1
	}
	defer store.DB.Close()

	auditLogger, err := audit.New(store.DB)
	if err != nil {
		fmt.Fprintf(stderr, "recover-admin: cannot open audit log: %v\n", err)
		return 1
	}
	encKey, _ := secrets.KeyFromEnv()
	authService := auth.NewService(store.DB, auditLogger, dataDir, encKey)

	if *list {
		return listAdmins(authService, stdout, stderr)
	}

	bg := breakGlassContext()
	target := strings.ToLower(strings.TrimSpace(*email))

	if !*yes && isKnownAdmin(authService, target) {
		fmt.Fprintf(stdout, "This resets the sign-in of admin %s:\n", target)
		fmt.Fprintln(stdout, "  - removes two-factor authentication and recovery codes")
		fmt.Fprintln(stdout, "  - sets a temporary password that must be changed at the next sign-in")
		fmt.Fprintln(stdout, "  - reactivates the account and signs out every session")
		fmt.Fprintln(stdout, "  - is recorded in the audit log")
		fmt.Fprint(stdout, "Type the email again to confirm: ")
		answer, _ := bufio.NewReader(stdin).ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != target {
			authService.LogBreakGlassAborted(target, bg)
			fmt.Fprintln(stderr, "recover-admin: confirmation did not match; nothing was changed")
			return 1
		}
	}

	res, err := authService.BreakGlassResetAdmin(target, bg)
	switch {
	case errors.Is(err, auth.ErrBreakGlassUserNotFound):
		fmt.Fprintf(stderr, "recover-admin: no account with email %q (see --list)\n", target)
		return 1
	case errors.Is(err, auth.ErrBreakGlassNotAdmin):
		fmt.Fprintf(stderr, "recover-admin: %s is not an admin. Normal users are reset from Admin → Users.\n", target)
		return 1
	case err != nil:
		fmt.Fprintf(stderr, "recover-admin: %v\n", err)
		return 1
	}

	alertBreakGlass(store, auditLogger, encKey, res, bg)

	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "Reset %s.\n", res.Email)
	if len(res.Cleared) > 0 {
		fmt.Fprintf(stdout, "Removed: %s\n", strings.Join(res.Cleared, ", "))
	}
	if res.Reactivated {
		fmt.Fprintln(stdout, "The account was inactive and has been reactivated.")
	}
	fmt.Fprintf(stdout, "Signed out %d session(s).\n", res.SessionsRevoked)
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "Temporary password (shown once): %s\n", res.TemporaryPassword)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Sign in with it, choose a new password, and set up two-factor authentication again.")
	return 0
}

func listAdmins(svc *auth.Service, stdout, stderr io.Writer) int {
	admins, err := svc.ListAdmins()
	if err != nil {
		fmt.Fprintf(stderr, "recover-admin: cannot list admins: %v\n", err)
		return 1
	}
	if len(admins) == 0 {
		fmt.Fprintln(stdout, "No admin accounts. Restarting the server creates the bootstrap admin.")
		return 0
	}
	tw := tabwriter.NewWriter(stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "EMAIL\tNAME\tACTIVE\t2FA")
	for _, a := range admins {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", a.Email, a.DisplayName, yesNo(a.IsActive), yesNo(a.TwoFactorEnabled))
	}
	tw.Flush()
	return 0
}

func isKnownAdmin(svc *auth.Service, email string) bool {
	admins, err := svc.ListAdmins()
	if err != nil {
		return false
	}
	for _, a := range admins {
		if a.Email == email {
			return true
		}
	}
	return false
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func breakGlassContext() auth.BreakGlassContext {
	bg := auth.BreakGlassContext{}
	bg.Hostname, _ = os.Hostname()
	if u, err := user.Current(); err == nil {
		bg.OSUser = u.Username
	}
	return bg
}

// alertBreakGlass tells the configured integrations that break-glass was used,
// so a legitimate or hostile run does not go unnoticed. It is best effort:
// a failed delivery is audited as notification_failed and the reset stands.
func alertBreakGlass(store *storage.Storage, auditLogger *audit.Logger, encKey []byte, res *auth.BreakGlassResult, bg auth.BreakGlassContext) {
	notifier := notify.NewService(notify.NewSettingsStore(store.DB, encKey), auditLogger)
	locale := i18n.ReadDefaultLocale(store.DB)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	notifier.Dispatch(ctx, notify.Event{
		Type:  notify.EventBreakGlassUsed,
		Title: i18n.T(locale, "notify.break_glass.title", nil),
		Message: i18n.T(locale, "notify.break_glass.message", map[string]string{
			"email":    res.Email,
			"hostname": bg.Hostname,
		}),
		Extra: map[string]any{
			"user_id":          res.UserID,
			"reactivated":      res.Reactivated,
			"sessions_revoked": res.SessionsRevoked,
		},
	})
}
