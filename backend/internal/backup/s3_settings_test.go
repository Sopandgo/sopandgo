package backup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

var testS3Key = []byte("01234567890123456789012345678901")

func validS3Input() SaveS3Input {
	return SaveS3Input{
		Enabled:         true,
		Bucket:          "lab-backups",
		Region:          "eu-central-1",
		KeyPrefix:       "/lab/",
		AccessKeyID:     "AKIDEXAMPLE",
		SecretAccessKey: "secret-1",
		Interval:        "6h",
		RetentionMax:    7,
		RetentionDays:   14,
	}
}

func TestS3SettingsStore_SaveAndLoad(t *testing.T) {
	_, store, _ := newTestService(t)
	s := NewS3SettingsStore(store.DB, testS3Key)

	pub, err := s.GetPublic()
	if err != nil {
		t.Fatal(err)
	}
	if pub.Configured || pub.Enabled || pub.Interval != "24h0m0s" || pub.RetentionMax != 14 || pub.RetentionDays != 30 {
		t.Fatalf("defaults: got %+v", pub)
	}

	if err := s.Save(validS3Input()); err != nil {
		t.Fatal(err)
	}
	pub, err = s.GetPublic()
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Configured || !pub.SecretConfigured || pub.KeyPrefix != "lab" || pub.Interval != "6h0m0s" {
		t.Fatalf("after save: got %+v", pub)
	}
	cfg, err := s.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled || cfg.StaticSecret != "secret-1" || cfg.RootPrefix != "lab" || cfg.Interval.String() != "6h0m0s" {
		t.Fatalf("config: got %+v", cfg)
	}

	t.Run("blank secret keeps the stored one", func(t *testing.T) {
		in := validS3Input()
		in.SecretAccessKey = ""
		in.Enabled = false
		if err := s.Save(in); err != nil {
			t.Fatal(err)
		}
		cfg, err := s.LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Enabled || cfg.StaticSecret != "secret-1" {
			t.Fatalf("got %+v", cfg)
		}
	})

	t.Run("new access key ID needs its secret", func(t *testing.T) {
		in := validS3Input()
		in.AccessKeyID = "AKIDOTHER"
		in.SecretAccessKey = ""
		if err := s.Save(in); !errors.Is(err, ErrInvalidS3Settings) {
			t.Fatalf("want ErrInvalidS3Settings, got %v", err)
		}
	})

	t.Run("blank access key ID uses the default chain", func(t *testing.T) {
		in := validS3Input()
		in.AccessKeyID = ""
		in.SecretAccessKey = ""
		if err := s.Save(in); err != nil {
			t.Fatal(err)
		}
		pub, _ := s.GetPublic()
		cfg, _ := s.LoadConfig()
		if pub.SecretConfigured || cfg.StaticKey != "" || cfg.StaticSecret != "" {
			t.Fatalf("got %+v / %+v", pub, cfg)
		}
	})
}

func TestS3SettingsStore_Validation(t *testing.T) {
	_, store, _ := newTestService(t)
	s := NewS3SettingsStore(store.DB, testS3Key)

	cases := []struct {
		name string
		edit func(*SaveS3Input)
	}{
		{"no bucket while on", func(in *SaveS3Input) { in.Bucket = "" }},
		{"bucket is a path", func(in *SaveS3Input) { in.Bucket = "lab/backups" }},
		{"bad endpoint", func(in *SaveS3Input) { in.Endpoint = "minio:9000" }},
		{"bad interval", func(in *SaveS3Input) { in.Interval = "daily" }},
		{"interval too short", func(in *SaveS3Input) { in.Interval = "30s" }},
		{"negative retention", func(in *SaveS3Input) { in.RetentionDays = -1 }},
		{"secret without key ID", func(in *SaveS3Input) { in.AccessKeyID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validS3Input()
			tc.edit(&in)
			if err := s.Save(in); !errors.Is(err, ErrInvalidS3Settings) {
				t.Fatalf("want ErrInvalidS3Settings, got %v", err)
			}
		})
	}

	t.Run("off without bucket is fine", func(t *testing.T) {
		if err := s.Save(SaveS3Input{}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("secret needs the encryption key", func(t *testing.T) {
		noKey := NewS3SettingsStore(store.DB, nil)
		if err := noKey.Save(validS3Input()); !errors.Is(err, ErrS3KeyMissing) {
			t.Fatalf("want ErrS3KeyMissing, got %v", err)
		}
	})
}

// A restore brings back the archive's data but keeps this instance's S3 settings.
func TestRestore_KeepsCurrentS3Settings(t *testing.T) {
	for _, tc := range []struct {
		name       string
		liveBucket string // "" = no settings row on the live instance
	}{
		{"live settings replace the archive's", "new-bucket"},
		{"no live settings removes the archive's", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			store, err := storage.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			svc := NewService(store.DB, dir, "t", storage.LatestSchemaVersion())
			settings := NewS3SettingsStore(store.DB, testS3Key)

			old := validS3Input()
			old.Bucket = "old-bucket"
			if err := settings.Save(old); err != nil {
				t.Fatal(err)
			}
			zipPath, _, _, err := svc.ExportZip()
			if err != nil {
				t.Fatal(err)
			}
			archive := filepath.Join(dir, "snapshot.zip")
			copyZipToFile(t, zipPath, archive)
			_ = os.RemoveAll(filepath.Dir(zipPath))

			if tc.liveBucket == "" {
				if _, err := store.DB.Exec(`DELETE FROM backup_s3_settings`); err != nil {
					t.Fatal(err)
				}
			} else {
				live := validS3Input()
				live.Bucket = tc.liveBucket
				if err := settings.Save(live); err != nil {
					t.Fatal(err)
				}
			}

			f, err := os.Open(archive)
			if err != nil {
				t.Fatal(err)
			}
			_, err = svc.StageImportApply(f, "RESTORE BACKUP")
			_ = f.Close()
			if err != nil {
				t.Fatal(err)
			}
			_ = store.DB.Close()

			if err := ApplyPendingRestoreAtStartup(dir); err != nil {
				t.Fatal(err)
			}
			store2, err := storage.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer store2.DB.Close()
			if err := ApplyCarriedS3Settings(store2.DB, dir); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(dir, carriedS3SettingsFile)); !os.IsNotExist(err) {
				t.Fatal("expected carry-over file removed")
			}

			pub, err := NewS3SettingsStore(store2.DB, testS3Key).GetPublic()
			if err != nil {
				t.Fatal(err)
			}
			if pub.Bucket != tc.liveBucket {
				t.Fatalf("bucket after restore = %q, want %q", pub.Bucket, tc.liveBucket)
			}
			if tc.liveBucket != "" {
				cfg, err := NewS3SettingsStore(store2.DB, testS3Key).LoadConfig()
				if err != nil || cfg.StaticSecret != "secret-1" {
					t.Fatalf("secret after restore: %v, %+v", err, cfg)
				}
			}
		})
	}
}

func TestApplyCarriedS3Settings_NoOpWithoutRestore(t *testing.T) {
	_, store, dir := newTestService(t)
	if err := ApplyCarriedS3Settings(store.DB, dir); err != nil {
		t.Fatal(err)
	}
}
