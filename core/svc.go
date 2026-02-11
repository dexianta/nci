package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SvcImpl struct {
	dbRepo DbRepo
}

func NewSvcImpl(dbRepo DbRepo) SvcImpl {
	return SvcImpl{dbRepo: dbRepo}
}

func (s SvcImpl) CloneRepo(repo, url string) error {
	if repo == "" {
		return errors.New("repo is empty")
	}
	// use clone helper
	err := CloneMirror(
		context.Background(),
		url,
		filepath.Join(Root, "repos", ToLocalRepo(repo)),
	) // just use convention, to get current path + "/repos")
	if err != nil {
		return err
	}

	return s.dbRepo.SaveCodeRepo(CodeRepo{
		Repo: repo,
		URL:  url,
	})
}

func (s SvcImpl) DeleteRepo(repo string) error {
	target := strings.TrimSpace(repo)
	if target == "" {
		return errors.New("repo is empty")
	}

	local := ToLocalRepo(target)
	paths := []string{
		filepath.Join(Root, "repos", local),
		filepath.Join(Root, "worktrees", local),
		filepath.Join(Root, "logs", local),
	}

	for _, p := range paths {
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("remove path %q: %w", p, err)
		}
	}

	if err := s.dbRepo.DeleteRepo(target); err != nil {
		return fmt.Errorf("delete repo from db: %w", err)
	}
	return nil
}
