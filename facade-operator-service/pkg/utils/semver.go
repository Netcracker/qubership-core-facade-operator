package utils

import (
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"strconv"
	"strings"
	"unicode"
)

type SemVer struct {
	Major int
	Minor int
	Patch int
}

func (v SemVer) IsNewerThanOrEqual(anotherVersion SemVer) bool {
	return v == anotherVersion || v.IsNewerThan(anotherVersion)
}

func (v SemVer) IsNewerThan(anotherVersion SemVer) bool {
	return v.Major > anotherVersion.Major ||
		(v.Major == anotherVersion.Major && v.Minor > anotherVersion.Minor) ||
		(v.Major == anotherVersion.Major && v.Minor == anotherVersion.Minor && v.Patch > anotherVersion.Patch)
}

func NewSemVer(version string) (SemVer, error) {
	version = sanitizeVersion(version)
	versions := strings.Split(version, ".")
	semVer := SemVer{}
	if len(versions) == 0 {
		return semVer, errs.NewError(customerrors.InitParamsValidationError, "Provided semver string does not have even major version", nil)
	}
	major, err := strconv.Atoi(versions[0])
	if err != nil {
		return semVer, errs.NewError(customerrors.InitParamsValidationError, "Could not parse major version from the provided semver", err)
	}
	semVer.Major = major

	if len(versions) > 1 {
		minor, hasSuffix := parseSubVersion(versions[1])
		semVer.Minor = minor

		if !hasSuffix && len(versions) > 2 {
			semVer.Patch, _ = parseSubVersion(versions[2])
		}
	}
	return semVer, nil
}

// parseSubVersion parses semver sub-version from a string and returns a digital version and a flag that indicates
// if there is also some string suffix after the digital version number.
//
// Examples:
//
// parseSubVersion("11") // returns 11, false
//
// parseSubVersion("11-SNAPSHOT") // returns 11, true
//
// parseSubVersion("SNAPSHOT") // returns 0, true
func parseSubVersion(version string) (int, bool) {
	intVersion, err := strconv.Atoi(version)
	if err == nil {
		return intVersion, false
	}
	return cutDigitsFromVersion(version), true
}

func cutDigitsFromVersion(version string) int {
	runes := []rune(version)
	firstNonDigitIndex := len(runes)
	for idx := 0; idx < len(runes); idx++ {
		if !unicode.IsDigit(runes[idx]) {
			firstNonDigitIndex = idx
			break
		}
	}
	if firstNonDigitIndex == 0 {
		return 0
	}
	intVersion, _ := strconv.Atoi(version[:firstNonDigitIndex])
	return intVersion
}

func sanitizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	return strings.TrimPrefix(version, "V")
}
