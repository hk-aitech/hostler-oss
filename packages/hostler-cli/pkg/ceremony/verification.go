// Package ceremony — post-registration verification (Phase 6/9).
// At the very end of the sprint:complete ceremony, immediately after
// KB/Task registrations, this verifies that the registered IDs actually
// exist as DB rows. Prevents the recurrence pattern where registration
// is only reported but no row is present.
// Verification is abstracted via a lookup so unit tests can run without
// a real DB by supplying a mock.
package ceremony

// RegistrationLookup looks up whether the given verification IDs exist.
// The real implementation wraps the DB store on the hostler CLI side;
// unit tests use a fake.
type RegistrationLookup interface {
	KBExists(id string) (bool, error)
	TaskExists(id string) (bool, error)
}

// VerificationResult accumulates the verification outcome.
// Passed is true only when every missing list is empty. ChecksError is
// not part of this struct; lookup errors are returned via the function
// signature, separately from the verification result.
type VerificationResult struct {
	Passed       bool     `json:"passed"`
	MissingKBs   []string `json:"missing_kbs"`
	MissingTasks []string `json:"missing_tasks"`
	CheckedKBs   int      `json:"checked_kbs"`
	CheckedTasks int      `json:"checked_tasks"`
}

// VerifyRegistration verifies KB and Task IDs against the lookup. An
// empty slice means "nothing to check" and counts as passed; nil is
// treated as an empty slice.
// Lookup errors return immediately — partial verification results are
// not trustworthy. The caller decides whether to retry or fail-stop.
func VerifyRegistration(kbIDs, taskIDs []string, lookup RegistrationLookup) (*VerificationResult, error) {
	res := &VerificationResult{
		MissingKBs:   []string{},
		MissingTasks: []string{},
		CheckedKBs:   len(kbIDs),
		CheckedTasks: len(taskIDs),
	}

	for _, id := range kbIDs {
		if id == "" {
			continue
		}
		ok, err := lookup.KBExists(id)
		if err != nil {
			return nil, err
		}
		if !ok {
			res.MissingKBs = append(res.MissingKBs, id)
		}
	}

	for _, id := range taskIDs {
		if id == "" {
			continue
		}
		ok, err := lookup.TaskExists(id)
		if err != nil {
			return nil, err
		}
		if !ok {
			res.MissingTasks = append(res.MissingTasks, id)
		}
	}

	res.Passed = len(res.MissingKBs) == 0 && len(res.MissingTasks) == 0
	return res, nil
}
