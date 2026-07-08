package mirror

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	transporthttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/sirupsen/logrus"

	"github.com/asano69/hatchards/internal/errs"
)

// Connection は変更なし
type Connection struct {
	Name      string
	RemoteURL string
	Username  string
	LocalPath string
}

func isGitRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

// Sync brings LocalPath up to date with RemoteURL.
//
// token may be empty when mirroring a public repository that requires no
// authentication. In that case, no auth credential is attached at all —
// go-git accepts a nil AuthMethod and performs an anonymous fetch/clone.
func Sync(conn Connection, token []byte) error {
	auth := buildAuth(conn.Username, token)

	log := logrus.WithFields(logrus.Fields{
		"connection": conn.Name,
		"remote_url": conn.RemoteURL,
		"username":   conn.Username,
		"local_path": conn.LocalPath,
		"anonymous":  auth == nil,
	})

	info, statErr := os.Stat(conn.LocalPath)
	switch {
	case os.IsNotExist(statErr):
		log.Info("mirror sync: local_path does not exist, cloning")
		return clone(conn, auth, log)

	case statErr != nil:
		log.WithError(statErr).Error("mirror sync: failed to stat local_path")
		return errs.Newf("stat local path %s: %v", conn.LocalPath, statErr)

	case !info.IsDir():
		log.Error("mirror sync: local_path exists but is not a directory")
		return errs.Newf("local path %s exists but is not a directory", conn.LocalPath)

	case !isGitRepo(conn.LocalPath):
		log.Warn("mirror sync: local_path exists but has no .git, cloning into it")
		return clone(conn, auth, log)

	default:
		log.Info("mirror sync: local_path is an existing git repo, pulling")
		return pull(conn, auth, log)
	}
}

// buildAuth returns a BasicAuth credential, or nil when both username and
// token are empty. A nil AuthMethod tells go-git to attempt an
// unauthenticated (anonymous) operation, which is required for mirroring
// public repositories that reject requests carrying an empty-but-present
// Basic auth header.
func buildAuth(username string, token []byte) transport.AuthMethod {
	if username == "" && len(token) == 0 {
		return nil
	}
	return &transporthttp.BasicAuth{Username: username, Password: string(token)}
}

func clone(conn Connection, auth transport.AuthMethod, log *logrus.Entry) error {
	_, err := git.PlainClone(conn.LocalPath, false, &git.CloneOptions{
		URL:  conn.RemoteURL,
		Auth: auth,
	})
	if err != nil {
		log.WithError(err).Error("mirror sync: clone failed")
		return errs.Newf("clone %q: %v", conn.Name, err)
	}
	log.Info("mirror sync: clone succeeded")
	return nil
}

func pull(conn Connection, auth transport.AuthMethod, log *logrus.Entry) error {
	repo, err := git.PlainOpen(conn.LocalPath)
	if err != nil {
		log.WithError(err).Error("mirror sync: PlainOpen failed")
		return errs.Newf("open local repo for %q: %v", conn.Name, err)
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		log.WithError(err).Error("mirror sync: could not read origin remote")
		return errs.Newf("read origin remote for %q: %v", conn.Name, err)
	}
	actualURLs := remote.Config().URLs
	if len(actualURLs) == 0 || actualURLs[0] != conn.RemoteURL {
		log.WithField("origin_url", actualURLs).
			Error("mirror sync: local_path's origin does not match this connection's remote_url — local_path is likely shared by another connection")
		return errs.Newf(
			"local path %s is a git repo whose origin (%v) does not match this connection's remote_url (%s) — check that local_path is not shared with another connection",
			conn.LocalPath, actualURLs, conn.RemoteURL,
		)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		log.WithError(err).Error("mirror sync: get worktree failed")
		return errs.Newf("get worktree for %q: %v", conn.Name, err)
	}
	err = worktree.Pull(&git.PullOptions{RemoteName: "origin", Auth: auth})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		log.WithError(err).Error("mirror sync: pull failed")
		return errs.Newf("pull %q: %v", conn.Name, err)
	}
	if errors.Is(err, git.NoErrAlreadyUpToDate) {
		log.Info("mirror sync: pull succeeded (already up to date)")
	} else {
		log.Info("mirror sync: pull succeeded")
	}
	return nil
}
