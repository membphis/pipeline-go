package handoff

import (
	"os"
	"path/filepath"
	"strings"
)

type Note struct {
	Source  string
	Content string
}

func Collect(root string) ([]Note, error) {
	var notes []Note
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return notes, nil
	}
	err = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		if strings.ToUpper(filepath.Base(path)) == "HANDOFF.MD" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			notes = append(notes, Note{
				Source:  path,
				Content: string(data),
			})
		}
		return nil
	})
	return notes, err
}

func FormatNotes(notes []Note) string {
	if len(notes) == 0 {
		return ""
	}
	var parts []string
	for _, n := range notes {
		parts = append(parts, "## Handoff: "+n.Source+"\n\n"+n.Content)
	}
	return strings.Join(parts, "\n\n---\n\n")
}
