package k8s

import (
	"os"
	"strings"
	"testing"

	pkggodevclient "github.com/guseggert/pkggodev-client"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

var (
	acceptedLicenses = map[string]struct{}{
		"MIT":          {},
		"Apache-2.0":   {},
		"BSD-3-Clause": {},
		"BSD-2-Clause": {},
		"ISC":          {},
		"CC-BY-SA-4.0": {},
		"MPL-2.0":      {},
	}

	knownUndectedLicenses = map[string]string{}
)

func TestLicenses(t *testing.T) {
	b, err := os.ReadFile("go.mod")
	require.NoError(t, err)
	file, err := modfile.Parse("go.mod", b, nil)
	require.NoError(t, err)
	client := pkggodevclient.New()
	for _, req := range file.Require {
		pkg, err := client.DescribePackage(pkggodevclient.DescribePackageRequest{
			Package: req.Mod.Path,
		})
		require.NoError(t, err, req.Mod.Path)
		licences := strings.Split(pkg.License, ",")
		for _, license := range licences {
			license = strings.TrimSpace(license)
			if license == "None detected" {
				if known, ok := knownUndectedLicenses[req.Mod.String()]; ok {
					license = known
				}
			}
			if _, ok := acceptedLicenses[license]; !ok {
				t.Errorf("dependency %s is using unexpected license %s. Check that this license complies with MIT in which grafana-cloud-operator is released and update the checks accordingly or change dependency", req.Mod, license)
			}
		}
	}
}
