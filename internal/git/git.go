package git

import (
	"io"
	"os"

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

	// Get the remote URL to re-clone
	remote, err := repo.Remote("origin")
	if err != nil {
		return false, err
	}
	url := remote.Config().URLs[0]

	// Get old HEAD to compare later
	oldRef, oldErr := repo.Head()

	// Re-clone (depth=1 is fast, avoids go-git Pull issues with file modes on Windows)
	os.RemoveAll(dir)
	if err := Clone(url, dir, progress); err != nil {
		return false, err
	}

	if oldErr != nil {
		return true, nil
	}

	repo, err = gogit.PlainOpen(dir)
	if err != nil {
		return true, nil
	}
	newRef, err := repo.Head()
	if err != nil {
		return true, nil
	}
	return oldRef.Hash() != newRef.Hash(), nil
}
