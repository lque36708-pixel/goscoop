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

func Pull(dir string, progress io.Writer) error {
	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	// Discard any local changes before pulling
	ref, err := repo.Head()
	if err != nil {
		return err
	}
	_ = wt.Reset(&gogit.ResetOptions{
		Mode:   gogit.HardReset,
		Commit: ref.Hash(),
	})

	err = wt.Pull(&gogit.PullOptions{
		Progress: progress,
	})
	if err == gogit.NoErrAlreadyUpToDate {
		return nil
	}
	return err
}
