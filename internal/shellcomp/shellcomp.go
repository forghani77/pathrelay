// Package shellcomp installs and removes shell completion scripts.
package shellcomp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const BinaryName = "pathrelay"

type spec struct {
	shell string
	dir   string
	file  string
	gen   func(*cobra.Command, io.Writer) error
}

// Specs describes where each shell's completion script lives.
func Specs() []spec {
	return []spec{
		{
			shell: "bash",
			dir:   "/etc/bash_completion.d",
			file:  BinaryName,
			gen:   func(c *cobra.Command, w io.Writer) error { return c.GenBashCompletionV2(w, true) },
		},
		{
			shell: "zsh",
			dir:   "/usr/local/share/zsh/site-functions",
			file:  "_" + BinaryName,
			gen:   func(c *cobra.Command, w io.Writer) error { return c.GenZshCompletion(w) },
		},
		{
			shell: "fish",
			dir:   "/etc/fish/completions",
			file:  BinaryName + ".fish",
			gen:   func(c *cobra.Command, w io.Writer) error { return c.GenFishCompletion(w, true) },
		},
	}
}

// Install writes a completion script for every shell whose completion
// directory exists; shells without one are skipped and reported.
func Install(root *cobra.Command) (installed, skipped []string, err error) {
	for _, s := range Specs() {
		if fi, statErr := os.Stat(s.dir); statErr != nil || !fi.IsDir() {
			skipped = append(skipped, s.shell)
			continue
		}
		var buf bytes.Buffer
		if err := s.gen(root, &buf); err != nil {
			return nil, nil, fmt.Errorf("generating %s completion: %w", s.shell, err)
		}
		path := filepath.Join(s.dir, s.file)
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			return nil, nil, fmt.Errorf("writing %s: %w", path, err)
		}
		installed = append(installed, path)
	}
	return installed, skipped, nil
}

// Remove deletes completion files that exist; missing ones are reported in
// the second slice rather than treated as errors.
func Remove() (removed, missing []string, err error) {
	for _, s := range Specs() {
		path := filepath.Join(s.dir, s.file)
		if _, statErr := os.Stat(path); statErr != nil {
			missing = append(missing, path)
			continue
		}
		if rmErr := os.Remove(path); rmErr != nil {
			return removed, missing, fmt.Errorf("removing %s: %w", path, rmErr)
		}
		removed = append(removed, path)
	}
	return removed, missing, nil
}
