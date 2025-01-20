package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

type testpair struct {
	beforeFiles   map[string]string
	beforeVersion string

	afterFiles   map[string]string
	afterVersion string
}

func TestMain(t *testing.T) {
	t.Parallel()

	assert := NewGomegaWithT(t)

	tests := []testpair{
		{
			beforeFiles:   map[string]string{},
			beforeVersion: "0.0.1",
			afterFiles:    map[string]string{},
			afterVersion:  "0.0.2",
		},
		{
			beforeFiles: map[string]string{
				"test.go": "package main\ntype Test struct{}",
			},
			beforeVersion: "0.1.0",
			afterFiles: map[string]string{
				"test.go": "package main\ntype Test struct{}",
			},
			afterVersion: "0.1.1",
		},
	}

	path, err := gexec.Build("github.com/jtarchie/semtype")
	assert.Expect(err).NotTo(HaveOccurred())
	defer gexec.CleanupBuildArtifacts()

	for index, test := range tests {
		t.Run(fmt.Sprintf("test %d", index), func(t *testing.T) {
			assert := NewGomegaWithT(t)

			dir, err := os.MkdirTemp("", "")
			assert.Expect(err).NotTo(HaveOccurred())

			for filename, contents := range test.beforeFiles {
				fullPath := filepath.Join(dir, filename)

				dir := filepath.Dir(fullPath)
				err := os.MkdirAll(dir, os.ModePerm)
				assert.Expect(err).NotTo(HaveOccurred())

				err = os.WriteFile(fullPath, []byte(contents), 0644)
				assert.Expect(err).NotTo(HaveOccurred())
			}

			output := gbytes.NewBuffer()
			session, err := gexec.Start(exec.Command(path, "-dir", dir), output, output)
			assert.Expect(err).NotTo(HaveOccurred())
			assert.Eventually(session).Should(gexec.Exit(0))
			assert.Expect(output).To(gbytes.Say(test.beforeVersion))

			for filename, contents := range test.afterFiles {
				fullPath := filepath.Join(dir, filename)

				dir := filepath.Dir(fullPath)
				err := os.MkdirAll(dir, os.ModePerm)
				assert.Expect(err).NotTo(HaveOccurred())

				err = os.WriteFile(fullPath, []byte(contents), 0644)
				assert.Expect(err).NotTo(HaveOccurred())
			}

			output = gbytes.NewBuffer()
			session, err = gexec.Start(exec.Command(path, "-dir", dir), output, output)
			assert.Expect(err).NotTo(HaveOccurred())
			assert.Eventually(session).Should(gexec.Exit(0))
			assert.Expect(output).To(gbytes.Say(test.afterVersion))
		})

	}
}
