package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadActivationMailTemplateConfig(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "activation-mail-templates.toml")

	content := `TemplateFile = "AktivierungsEmail-template.html"

[[InlinePictures]]
ContentId = "logo"
Filepath = "logo.png"
`
	require.NoError(t, os.WriteFile(tomlPath, []byte(content), 0o600))

	config, err := ReadActivationMailTemplateConfig(tomlPath)
	require.NoError(t, err)
	assert.Equal(t, "AktivierungsEmail-template.html", config.TemplateFile)
	assert.Len(t, config.InlinePictures, 1)
}

func TestReadActivationMailTemplateConfig_MissingFile(t *testing.T) {
	_, err := ReadActivationMailTemplateConfig(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	assert.Error(t, err)
}
