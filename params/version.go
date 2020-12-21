package params

import (
	"fmt"
)

// version parts
const (
	VersionMajor = 0  // Major version component of the current release
	VersionMinor = 3  // Minor version component of the current release
	VersionPatch = 2  // Patch version component of the current release
	VersionMeta  = "" // Version metadata to append to the version string
)

const (
	versionStable = "stable"
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

// ArchiveVersion holds the textual version string used for Geth archives.
// e.g. "1.8.11-dea1ce05" for stable releases, or
//      "1.8.13-unstable-21c059b6" for unstable releases
func ArchiveVersion(gitCommit string) string {
	vsn := Version
	if VersionMeta != versionStable {
		vsn += "-" + VersionMeta
	}
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	return vsn
}

// VersionWithCommit add git commit and data to version.
func VersionWithCommit(gitCommit, gitDate string) string {
	vsn := VersionWithMeta
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	if (VersionMeta != versionStable) && (gitDate != "") {
		vsn += "-" + gitDate
	}
	return vsn
}
