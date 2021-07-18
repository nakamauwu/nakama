package http

import (
	"regexp"
	"strconv"
)

// Don’t send `SameSite=None` to known incompatible clients.
func shouldSendSameSiteNone(useragent string) (bool, error) {
	sameSiteNoneIncompatible, err := isSameSiteNoneIncompatible(useragent)
	if err != nil {
		return false, nil
	}
	return !sameSiteNoneIncompatible, nil
}

// Classes of browsers known to be incompatible.
func isSameSiteNoneIncompatible(useragent string) (bool, error) {
	sameSiteBug, err := hasWebKitSameSiteBug(useragent)
	if err != nil {
		return false, nil
	}
	cookies, err := dropsUnrecognizedSameSiteCookies(useragent)
	if err != nil {
		return false, nil
	}
	return sameSiteBug || cookies, nil
}

func hasWebKitSameSiteBug(useragent string) (bool, error) {
	isIosBVersion, err := isIosVersion(12, useragent)
	if err != nil {
		return false, nil
	}
	isMacOsVersion, err := isMacosxVersion(10, 14, useragent)
	if err != nil {
		return false, nil
	}
	safari, err := isSafari(useragent)
	macEmbeddedBrowser, err := isMacEmbeddedBrowser(useragent)
	return isIosBVersion || (isMacOsVersion && (safari || macEmbeddedBrowser)), nil
}

func dropsUnrecognizedSameSiteCookies(useragent string) (bool, error) {
	browser, err := isUcBrowser(useragent)
	if err != nil {
		return false, nil
	}
	if browser {
		ucBrowserVersionAtLeast, err := isUcBrowserVersionAtLeast(12, 13, 2, useragent)
		if err != nil {
			return false, nil
		}
		return !ucBrowserVersionAtLeast, nil
	}
	chromiumBased, err := isChromiumBased(useragent)
	if err != nil {
		return false, nil
	}
	chromiumVersionAtLeast, err := isChromiumVersionAtLeast(51, useragent)
	if err != nil {
		return false, nil
	}
	versionAtLeast, err := isChromiumVersionAtLeast(67, useragent)
	if err != nil {
		return false, nil
	}
	return chromiumBased && chromiumVersionAtLeast && !versionAtLeast, nil
}

// Regex parsing of User-Agent string. (See note above!)
// Regex parsing of User-Agent string. (See note above!)
func isIosVersion(major int, useragent string) (bool, error) {
	regex := "\\(iP.+; CPU .*OS (\\d+)[_\\d]*.*\\) AppleWebKit\\/"
	r, err := regexp.Compile(regex)
	if err != nil {
		return false, nil
	}
	matches := r.FindAllStringSubmatch(useragent, -1)
	if matches == nil {
		return false, nil
	}
	majorVersion, err := strconv.Atoi(matches[0][1])
	if err != nil {
		return false, err
	}
	return majorVersion == major, nil
}

func isMacosxVersion(major int, minor int, useragent string) (bool, error) {
	regex := "\\(Macintosh;.*Mac OS X (\\d+)_(\\d+)[_\\d]*.*\\) AppleWebKit\\/"
	r, err := regexp.Compile(regex)
	if err != nil {
		return false, nil
	}
	// Extract digits from first and second capturing groups.
	matches := r.FindAllStringSubmatch(useragent, -1)
	if matches == nil {
		return false, nil
	}
	majorVersion, err := strconv.Atoi(matches[0][1])
	if err != nil {
		return false, err
	}
	minorVersion, err := strconv.Atoi(matches[0][2])
	if err != nil {
		return false, err
	}

	return majorVersion == major && minorVersion == minor, nil
}

func isSafari(useragent string) (bool, error) {
	regex := "Version\\/.* Safari\\/"
	r, err := regexp.Compile(regex)
	if err != nil {
		return false, nil
	}
	isChromiumBased, err := isChromiumBased(useragent)
	if err != nil {
		return false, err
	}
	return r.MatchString(useragent) && !isChromiumBased, nil
}

func isMacEmbeddedBrowser(useragent string) (bool, error) {
	regex := "^Mozilla\\/[\\.\\d]+ \\(Macintosh;.*Mac OS X [_\\d]+\\)" + "AppleWebKit\\/[\\.\\d]+ \\(KHTML, like Gecko\\)$"
	r, err := regexp.Compile(regex)
	if err != nil {
		return false, err
	}
	return r.MatchString(useragent), nil
}
func isChromiumBased(useragent string) (bool, error) {
	regex := "Chrom(e|ium)"
	r, err := regexp.Compile(regex)
	if err != nil {
		return false, err
	}
	return r.MatchString(useragent), nil
}
func isChromiumVersionAtLeast(major int, useragent string) (bool, error) {
	regex := "Chrom[^ \\/]+\\/(\\d+)[\\.\\d]* "
	r, err := regexp.Compile(regex)
	if err != nil {
		return false, err
	}
	// Extract digits from first capturing group.
	matches := r.FindAllStringSubmatch(useragent, -1)
	if matches == nil {
		return false, nil
	}
	majorVersion, err := strconv.Atoi(matches[0][1])
	return majorVersion >= major, err
}
func isUcBrowser(useragent string) (bool, error) {
	regex := "UCBrowser\\/"
	r, err := regexp.Compile(regex)
	if err != nil {
		return false, err
	}
	return r.MatchString(useragent), nil
}
func isUcBrowserVersionAtLeast(major int, minor int, build int, useragent string) (bool, error) {
	regex := "UCBrowser\\/(\\d+)\\.(\\d+)\\.(\\d+)[\\.\\d]* "
	r, err := regexp.Compile(regex)
	if err != nil {
		return false, err
	}
	matches := r.FindAllStringSubmatch(useragent, -1)
	if matches == nil {
		return false, nil
	}
	majorVersion, err := strconv.Atoi(matches[0][1])
	if err != nil {
		return false, err
	}
	minorVersion, err := strconv.Atoi(matches[0][2])
	if err != nil {
		return false, err
	}
	buildVersion, err := strconv.Atoi(matches[0][3])
	if err != nil {
		return false, err
	}
	// Extract digits from three capturing groups.
	if majorVersion != major {
		return majorVersion > major, nil
	}
	if minorVersion != minor {
		return minorVersion > minor, nil
	}
	return buildVersion >= build, nil
}
