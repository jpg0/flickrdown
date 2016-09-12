package processing

import (
	"time"
	"strings"
)

type TaggedFile interface {
	Filepath() string
	Name() string
	Keywords() Keywords
	DateTaken() time.Time
	StringTag(name string) string
	ReplaceStringTag(old string, new string) error
}

type Keywords interface {
	All() *TagSet
	Replace(old string, new string) error
}

type KeywordsHelper struct {
	Keywords
}

func ValuesByPrefix(k Keywords, prefix string) []string {
	values := make([]string, 0)
	for _, v := range k.All().Slice() {
		if strings.HasPrefix(v, prefix) {
			values = append(values, v[len(prefix):])
		}
	}
	return values
}