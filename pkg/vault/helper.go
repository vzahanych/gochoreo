package vault

import (
	"crypto/x509"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// appendCertsFromDir loads all certificate files from a directory
func appendCertsFromDir(certPool *x509.CertPool, certDir string) error {
	return filepath.WalkDir(certDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !isCertFile(d.Name()) {
			return nil
		}

		certData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read certificate file %s: %w", path, err)
		}

		if !certPool.AppendCertsFromPEM(certData) {
			return fmt.Errorf("failed to parse certificate in file %s", path)
		}

		return nil
	})
}

// isCertFile checks if a filename appears to be a certificate file
func isCertFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".pem" || ext == ".crt" || ext == ".cer" || ext == ".cert"
}
