package auth_test

import (
	"errors"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/errkind"
	"github.com/sopandgo/sopandgo/backend/internal/imageutil"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_AvatarErrorKinds(t *testing.T) {
	env := testenv.New(t)
	userID := "avatar-kinds-user"
	env.SeedTestUser(t, userID, "avatar-kinds@demo.local", auth.RoleViewer)

	t.Run("removing a missing avatar is not found", func(t *testing.T) {
		err := env.AuthService.RemoveAvatar(userID, &userID)
		if !errors.Is(err, auth.ErrNoAvatar) || !errors.Is(err, errkind.NotFound) {
			t.Fatalf("want no avatar, got %v", err)
		}
	})

	t.Run("removing the avatar of an unknown user is not found", func(t *testing.T) {
		missing := "no-such-user"
		err := env.AuthService.RemoveAvatar(missing, &missing)
		if !errors.Is(err, errkind.NotFound) || errors.Is(err, auth.ErrNoAvatar) {
			t.Fatalf("want a not found that isn't no avatar, got %v", err)
		}
	})

	t.Run("a missing avatar's path is not found", func(t *testing.T) {
		_, _, err := env.AuthService.AvatarAbsolutePath(userID, "sm")
		if !errors.Is(err, auth.ErrNoAvatar) || !errors.Is(err, errkind.NotFound) {
			t.Fatalf("want no avatar, got %v", err)
		}
	})

	t.Run("an unknown size is invalid", func(t *testing.T) {
		_, _, err := env.AuthService.AvatarAbsolutePath(userID, "huge")
		if !errors.Is(err, auth.ErrInvalidAvatarSize) || !errors.Is(err, errkind.Invalid) {
			t.Fatalf("want invalid size, got %v", err)
		}
	})

	t.Run("a non-image upload passes the image module's error through", func(t *testing.T) {
		err := env.AuthService.SetAvatar(userID, []byte("not an image"), &userID)
		if !errors.Is(err, imageutil.ErrInvalidImage) {
			t.Fatalf("want invalid image, got %v", err)
		}
	})

	t.Run("an unknown user is not found", func(t *testing.T) {
		_, err := env.AuthService.GetUserByID("no-such-user")
		if !errors.Is(err, errkind.NotFound) {
			t.Fatalf("want not found, got %v", err)
		}
	})
}
