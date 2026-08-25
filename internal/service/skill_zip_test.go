package service

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"aidevclub/internal/platform"
)

type zipFixture struct {
	name    string
	content string
	mode    os.FileMode
}

func makeSkillZip(t *testing.T, entries ...zipFixture) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, entry := range entries {
		h := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		if entry.mode != 0 {
			h.SetMode(entry.mode)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(entry.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func markZipEncrypted(data []byte) {
	for i := 0; i+4 <= len(data); i++ {
		switch binary.LittleEndian.Uint32(data[i:]) {
		case 0x04034b50:
			data[i+6] |= 0x01
		case 0x02014b50:
			data[i+8] |= 0x01
		}
	}
}

func markZipUnsupported(data []byte) {
	for i := 0; i+4 <= len(data); i++ {
		switch binary.LittleEndian.Uint32(data[i:]) {
		case 0x04034b50:
			binary.LittleEndian.PutUint16(data[i+8:], 99)
		case 0x02014b50:
			binary.LittleEndian.PutUint16(data[i+10:], 99)
		}
	}
}

func markZipDirectoryName(data []byte, ordinaryName, directoryName string) {
	if len(ordinaryName) != len(directoryName) {
		panic("ZIP entry name replacement must keep the same length")
	}
	for i := 0; i+len(ordinaryName) <= len(data); i++ {
		if string(data[i:i+len(ordinaryName)]) == ordinaryName {
			copy(data[i:i+len(directoryName)], directoryName)
		}
	}
}

func requireInvalidSkillZip(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, platform.ErrInvalidInput) {
		t.Fatalf("error = %v, want platform.ErrInvalidInput", err)
	}
}

func TestExtractSkillMDFindsRootAndNestedFile(t *testing.T) {
	for _, tc := range []struct {
		name    string
		entry   string
		content string
	}{
		{name: "root", entry: "SKILL.md", content: "# Root\nUse it."},
		{name: "nested", entry: "demo/SKILL.md", content: "# Demo\nUse it."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := makeSkillZip(t, zipFixture{name: tc.entry, content: tc.content})

			got, err := extractSkillMD(bytes.NewReader(data), int64(len(data)))

			if err != nil {
				t.Fatal(err)
			}
			if got != tc.content {
				t.Fatalf("SKILL.md = %q, want %q", got, tc.content)
			}
		})
	}
}

func TestExtractSkillMDRejectsUnsafeArchive(t *testing.T) {
	encrypted := makeSkillZip(t, zipFixture{name: "SKILL.md", content: "encrypted"})
	markZipEncrypted(encrypted)
	unsupported := makeSkillZip(t, zipFixture{name: "SKILL.md", content: "unsupported"})
	markZipUnsupported(unsupported)
	directorySymlink := makeSkillZip(t,
		zipFixture{name: "linkX", content: "target", mode: os.ModeSymlink | 0o777},
		zipFixture{name: "SKILL.md", content: "valid"},
	)
	markZipDirectoryName(directorySymlink, "linkX", "link/")
	directoryPayload := makeSkillZip(t,
		zipFixture{name: "payloadX", content: strings.Repeat("x", 10<<20+1)},
		zipFixture{name: "SKILL.md", content: "valid"},
	)
	markZipDirectoryName(directoryPayload, "payloadX", "payload/")

	tooManyEntries := make([]zipFixture, 0, 257)
	tooManyEntries = append(tooManyEntries, zipFixture{name: "SKILL.md", content: "valid"})
	for i := 1; i < 257; i++ {
		tooManyEntries = append(tooManyEntries, zipFixture{name: fmt.Sprintf("files/%03d.txt", i), content: "x"})
	}

	cases := []struct {
		name string
		data []byte
	}{
		{name: "missing skill file", data: makeSkillZip(t, zipFixture{name: "README.md", content: "missing"})},
		{name: "traversal", data: makeSkillZip(t, zipFixture{name: "../SKILL.md", content: "escape"})},
		{name: "absolute path", data: makeSkillZip(t, zipFixture{name: "/SKILL.md", content: "escape"})},
		{name: "windows drive relative path", data: makeSkillZip(t, zipFixture{name: "C:dir/SKILL.md", content: "escape"})},
		{name: "windows drive absolute path", data: makeSkillZip(t, zipFixture{name: "C:/SKILL.md", content: "escape"})},
		{name: "duplicate skill files", data: makeSkillZip(t,
			zipFixture{name: "a/SKILL.md", content: "one"},
			zipFixture{name: "b/SKILL.md", content: "two"},
		)},
		{name: "encrypted entry", data: encrypted},
		{name: "unsupported compression", data: unsupported},
		{name: "symlink", data: makeSkillZip(t, zipFixture{name: "SKILL.md", content: "target", mode: os.ModeSymlink | 0o777})},
		{name: "directory marked symlink", data: directorySymlink},
		{name: "directory payload bypass", data: directoryPayload},
		{name: "expanded archive is too large", data: makeSkillZip(t,
			zipFixture{name: "SKILL.md", content: "valid"},
			zipFixture{name: "large.txt", content: strings.Repeat("x", 10<<20+1)},
		)},
		{name: "too many entries", data: makeSkillZip(t, tooManyEntries...)},
		{name: "skill file is too large", data: makeSkillZip(t, zipFixture{name: "SKILL.md", content: strings.Repeat("x", 1<<20+1)})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := extractSkillMD(bytes.NewReader(tc.data), int64(len(tc.data)))
			requireInvalidSkillZip(t, err)
		})
	}
}
