package duplicates

import (
	"errors"
	"testing"

	"duplica-scan/src/internal/model"
)

func TestDetectGroupsBySizeAndHash(t *testing.T) {
	files := []model.FileMeta{
		{Name: "a.txt", Path: "a.txt", Size: 10},
		{Name: "b.txt", Path: "b.txt", Size: 10},
		{Name: "c.txt", Path: "c.txt", Size: 50},
		{Name: "d.txt", Path: "d.txt", Size: 10},
	}

	hashes := map[string]string{
		"a.txt": "same",
		"b.txt": "same",
		"d.txt": "other",
	}

	groups, errs := Detect(files, func(path string) (string, error) {
		v, ok := hashes[path]
		if !ok {
			return "", errors.New("unexpected path")
		}
		return v, nil
	}, nil)

	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d", len(errs))
	}
	if len(groups) != 1 {
		t.Fatalf("expected one duplicate group, got %d", len(groups))
	}
	if len(groups[0].Files) != 2 {
		t.Fatalf("expected two files in group, got %d", len(groups[0].Files))
	}
}
