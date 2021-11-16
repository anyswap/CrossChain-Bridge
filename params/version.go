// Package params provides common version info and config items.
package params

import (
	"fmt"
)

// version parts
const (
	VersionMajor = 0  // Major version component of the current release
	VersionMinor = 3  // Minor version component of the current release
	VersionPatch = 9  // Patch version component of the current release
	VersionMeta  = "" // Version metadata to append to the version string
)

// Version holds the textual version string.
var Version = func() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}()

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = func() string {
	v := Version
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

// VersionWithCommit add git commit and data to version.
func VersionWithCommit(gitCommit, gitDate string) string {
	vsn := Version
	if VersionMeta != "" {
		vsn += "-" + VersionMeta
	}
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	if (VersionMeta != "stable") && (gitDate != "") {
		vsn += "-" + gitDate
	}
	VersionWithMeta = vsn // update if more concrete
	return vsn
}
