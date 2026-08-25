package service

import (
	"archive/zip"
	"io"
	"os"
	"path"
	"strings"

	"aidevclub/internal/platform"
)

const (
	maxArchiveEntries = 256
	maxExpandedBytes  = 10 << 20
	maxSkillMDBytes   = 1 << 20
)

func extractSkillMD(r io.ReaderAt, size int64) (string, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil || len(zr.File) > maxArchiveEntries {
		return "", platform.ErrInvalidInput
	}

	var skillMD string
	var foundSkillMD bool
	var expanded int64
	for _, file := range zr.File {
		if !safeSkillArchivePath(file.Name) || file.Flags&0x1 != 0 || (file.Method != zip.Store && file.Method != zip.Deflate) {
			return "", platform.ErrInvalidInput
		}
		mode := file.Mode()
		if mode&(os.ModeType&^os.ModeDir) != 0 {
			return "", platform.ErrInvalidInput
		}
		if file.FileInfo().IsDir() {
			if file.CompressedSize64 != 0 || file.UncompressedSize64 != 0 {
				return "", platform.ErrInvalidInput
			}
			continue
		}
		if mode&os.ModeType != 0 {
			return "", platform.ErrInvalidInput
		}

		isSkillMD := path.Base(path.Clean(file.Name)) == "SKILL.md"
		limit := maxExpandedBytes - expanded
		if isSkillMD && limit > maxSkillMDBytes {
			limit = maxSkillMDBytes
		}
		if limit < 0 {
			return "", platform.ErrInvalidInput
		}
		reader, err := file.Open()
		if err != nil {
			return "", platform.ErrInvalidInput
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, limit+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || int64(len(data)) > limit {
			return "", platform.ErrInvalidInput
		}
		expanded += int64(len(data))

		if !isSkillMD {
			continue
		}
		if foundSkillMD || len(data) > maxSkillMDBytes {
			return "", platform.ErrInvalidInput
		}
		foundSkillMD = true
		skillMD = string(data)
	}
	if !foundSkillMD {
		return "", platform.ErrInvalidInput
	}
	return skillMD, nil
}

func safeSkillArchivePath(name string) bool {
	if name == "" || strings.Contains(name, "\\") || path.IsAbs(name) || hasWindowsDrivePrefix(name) {
		return false
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == ".." {
			return false
		}
	}
	return true
}

func hasWindowsDrivePrefix(name string) bool {
	if len(name) < 2 || name[1] != ':' {
		return false
	}
	return (name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')
}
