package cleanup

import (
	"os"
)

type Result struct {
	Path    string
	Deleted bool
	Err     error
}

// DeleteFiles removes selected files unless dryRun is enabled.
func DeleteFiles(paths []string, dryRun bool) []Result {
	results := make([]Result, 0, len(paths))
	for _, path := range paths {
		if dryRun {
			results = append(results, Result{
				Path:    path,
				Deleted: false,
				Err:     nil,
			})
			continue
		}

		err := os.Remove(path)
		results = append(results, Result{
			Path:    path,
			Deleted: err == nil,
			Err:     err,
		})
	}
	return results
}
