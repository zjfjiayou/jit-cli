package profile

import (
	"errors"
	"fmt"
	"regexp"
)

var patPattern = regexp.MustCompile(`^jit_pat_([A-Za-z0-9]+)_([A-Za-z0-9]+)$`)

var ErrInvalidPAT = errors.New("invalid pat format")

// PATParts is the parsed representation of jit_pat_{tokenId}_{secret}.
type PATParts struct {
	TokenID string
	Secret  string
}

func ParsePAT(token string) (PATParts, error) {
	matches := patPattern.FindStringSubmatch(token)
	if len(matches) != 3 {
		return PATParts{}, fmt.Errorf("%w: expected jit_pat_{tokenId}_{secret}", ErrInvalidPAT)
	}
	return PATParts{
		TokenID: matches[1],
		Secret:  matches[2],
	}, nil
}

func ValidatePAT(token string) error {
	_, err := ParsePAT(token)
	return err
}
