package session

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/camdenwithrow/redwood/internal/repository"
)

const readableNameLimit = 36

func Name(repo repository.Repository, worktree repository.Worktree) string {
	identity := worktree.Branch
	if identity == "" {
		identity = worktree.Commit
	}
	readable := slug(repo.Name + "-" + identity)
	if len(readable) > readableNameLimit {
		readable = strings.Trim(readable[:readableNameLimit], "-")
	}
	if readable == "" {
		readable = "worktree"
	}

	digest := sha256.Sum256([]byte(repo.GitDir + "\x00" + worktree.Path))
	hash := hex.EncodeToString(digest[:])[:12]
	return "rw-" + readable + "-" + hash
}

func slug(value string) string {
	var result strings.Builder
	separator := false
	for _, character := range strings.ToLower(value) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			result.WriteRune(character)
			separator = false
			continue
		}
		if result.Len() > 0 && !separator {
			result.WriteByte('-')
			separator = true
		}
	}

	return strings.Trim(result.String(), "-")
}
