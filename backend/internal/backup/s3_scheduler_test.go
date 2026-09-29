package backup

import (
	"testing"
	"time"
)

func TestAutomatedListPrefix(t *testing.T) {
	if got := automatedListPrefix(""); got != "automated/" {
		t.Fatalf("empty: got %q", got)
	}
	if got := automatedListPrefix("prod"); got != "prod/automated/" {
		t.Fatalf("prod: got %q", got)
	}
	if got := automatedListPrefix("/staging/"); got != "staging/automated/" {
		t.Fatalf("staging: got %q", got)
	}
}

func TestKeysToDeleteForRetention(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	o1 := s3ObjectMeta{key: "a.zip", t: now.Add(-48 * time.Hour)}
	o2 := s3ObjectMeta{key: "b.zip", t: now.Add(-25 * time.Hour)}
	o3 := s3ObjectMeta{key: "c.zip", t: now.Add(-1 * time.Hour)}
	o4 := s3ObjectMeta{key: "d.zip", t: now.Add(-30 * time.Minute)}

	t.Run("max only", func(t *testing.T) {
		objs := []s3ObjectMeta{o1, o2, o3, o4}
		del := keysToDeleteForRetention(objs, 2, 0, now)
		if len(del) != 2 {
			t.Fatalf("want 2 deletes, got %v", del)
		}
		want := map[string]bool{"a.zip": true, "b.zip": true}
		for _, k := range del {
			if !want[k] {
				t.Fatalf("unexpected delete %q", k)
			}
		}
	})

	t.Run("age only", func(t *testing.T) {
		objs := []s3ObjectMeta{o1, o2, o3, o4}
		del := keysToDeleteForRetention(objs, 0, 1, now)
		if len(del) != 2 {
			t.Fatalf("want 2 deletes (older than 1 day), got %v", del)
		}
	})

	t.Run("age then max", func(t *testing.T) {
		objs := []s3ObjectMeta{o1, o2, o3, o4}
		del := keysToDeleteForRetention(objs, 2, 1, now)
		// o1,o2 removed by age; survivors o3,o4 -> max 2 keeps both -> no count deletes
		if len(del) != 2 {
			t.Fatalf("want 2 deletes, got %v", del)
		}
	})

	t.Run("max after age leaves excess", func(t *testing.T) {
		objs := []s3ObjectMeta{o3, o4, {key: "e.zip", t: now.Add(-10 * time.Minute)}}
		del := keysToDeleteForRetention(objs, 1, 0, now)
		if len(del) != 2 {
			t.Fatalf("want 2 deletes, got %v", del)
		}
	})
}

func TestLoadS3SchedulerConfigFromEnv(t *testing.T) {
	t.Setenv("BACKUP_S3_ENABLED", "")
	cfg, err := LoadS3SchedulerConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled {
		t.Fatal("expected disabled")
	}
}
