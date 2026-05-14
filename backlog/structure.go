package backlog

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	configFileName        = ".am/config.yaml"
	indexFileName         = "index.md"
	velocityFileName      = "velocity.md"
	velocityDirectoryName = "velocity"
	archiveDirectoryName  = "archive"
	TagsDirectoryName     = "tags"
	TagsFileName          = "tags.md"
	usersDirectoryName    = "users"
	usersFileName         = "users.md"
	timelineFileName      = "timeline.md"
	timelineDirectoryName = "timeline"
)

var (
	ForbiddenBacklogNames = []string{velocityDirectoryName, archiveDirectoryName, TagsDirectoryName, usersDirectoryName, timelineDirectoryName}
	ForbiddenItemNames    = []string{archiveDirectoryName, "_priority", "_icebox"}
)

type BacklogsStructure struct {
	root string
}

func NewBacklogsStructure(root string) *BacklogsStructure {
	return &BacklogsStructure{root: root}
}

func (s *BacklogsStructure) Root() string {
	return s.root
}

func (s *BacklogsStructure) ConfigFile() string {
	return filepath.Join(s.root, configFileName)
}

func (s *BacklogsStructure) IndexFile() string {
	return filepath.Join(s.root, indexFileName)
}

func (s *BacklogsStructure) VelocityFile() string {
	return filepath.Join(s.root, velocityFileName)
}

func (s *BacklogsStructure) TagsFile() string {
	return filepath.Join(s.root, TagsFileName)
}

func (s *BacklogsStructure) UsersFile() string {
	return filepath.Join(s.root, usersFileName)
}

func (s *BacklogsStructure) TimelineFile() string {
	return filepath.Join(s.root, timelineFileName)
}

func (s *BacklogsStructure) UsersDirectory() string {
	return filepath.Join(s.root, usersDirectoryName)
}

func (s *BacklogsStructure) TagsDirectory() string {
	return filepath.Join(s.root, TagsDirectoryName)
}

func (s *BacklogsStructure) TimelineDirectory() string {
	return filepath.Join(s.root, timelineDirectoryName)
}

func (s *BacklogsStructure) BacklogDirs() ([]string, error) {
	infos, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.IsDir() || strings.HasPrefix(info.Name(), ".") || IsForbiddenBacklogName(info.Name()) {
			continue
		}
		result = append(result, filepath.Join(s.root, info.Name()))
	}
	sort.Strings(result)
	return result, nil
}
