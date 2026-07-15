package git

import (
	"io"

	gogit "github.com/go-git/go-git/v5"
)

func Clone(url, dir string, progress io.Writer) error {
	_, err := gogit.PlainClone(dir, false, &gogit.CloneOptions{
		URL:      url,
		Progress: progress,
		Depth:    1,
	})
	return err
}

func Pull(dir string, progress io.Writer) (bool, error) {
	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		return false, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return false, err
	}

	oldHash, err := repo.Head()
	if err != nil {
		return false, err
	}

	// Discard any local changes before pulling
	_ = wt.Reset(&gogit.ResetOptions{
		Mode:   gogit.HardReset,
		Commit: oldHash.Hash(),
	})

	err = wt.Pull(&gogit.PullOptions{
		Progress: progress,
	})
	if err == gogit.NoErrAlreadyUpToDate {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Check if HEAD actually changed (got new commits)
	newHash, err := repo.Head()
	if err != nil {
		return false, err
	}
	return oldHash.Hash() != newHash.Hash(), nil
}
