package selfupdate

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/mod/semver"
)

const (
	repoOwner = "Aspiand"
	repoName  = "uptime-go"
	apiURL    = "https://api.github.com/repos/%s/%s/releases/latest"
)

var (
	assetName  = fmt.Sprintf("uptime-go_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	releaseUrl = fmt.Sprintf(apiURL, repoOwner, repoName)
)

type gitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
		Digest      string `json:"digest"`
	} `json:"assets"`
}

func Run(currentVersion string, isDryRun, isForce bool) error {
	if !strings.HasPrefix(currentVersion, "v") {
		currentVersion = "v" + currentVersion
	}

	fmt.Println("Checking for new versions...")
	release, err := getLatestReleaseInfo()
	if err != nil {
		return fmt.Errorf("could not get latest release info: %w", err)
	}

	latestVersion := release.TagName
	if !semver.IsValid(currentVersion) || !semver.IsValid(latestVersion) {
		return fmt.Errorf("invalid version format. Current: %s, Latest: %s", currentVersion, release.TagName)
	}

	if semver.Compare(currentVersion, latestVersion) >= 0 {
		fmt.Printf("Current version (%s) is already the latest.\n", currentVersion)
		return nil
	}

	fmt.Printf("A new version %s is available! Current version is %s\n", latestVersion, currentVersion)

	var assetURL, expectedChecksum string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			assetURL = asset.DownloadURL
			if strings.HasPrefix(asset.Digest, "sha256:") {
				expectedChecksum = strings.TrimPrefix(asset.Digest, "sha256:")
			}
		}
	}

	if assetURL == "" {
		return fmt.Errorf("could not find asset for %s in release %s", assetName, latestVersion)
	}
	if expectedChecksum == "" {
		return fmt.Errorf("could not find checksum for %s in release %s", assetName, latestVersion)
	}

	fmt.Printf("Downloading new binary: %s\n", assetName)
	tmpFile, downloadedChecksum, err := downloadFile(assetURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	fmt.Printf("\nVerifying checksum...")
	if downloadedChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch! Expected %s, got %s", expectedChecksum, downloadedChecksum)
	}
	fmt.Println(" OK")

	fmt.Println("Extracting binary from tarball...")
	extractedBinaryPath, err := extractBinaryFromTarball(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}
	defer os.Remove(extractedBinaryPath)
	fmt.Println("Extraction successful.")

	if isDryRun {
		fmt.Println("\nDry run successful. Binary downloaded and verified.")
		fmt.Printf("The new binary is at: %s\n", extractedBinaryPath)
		fmt.Println("(This temporary file will be deleted upon exit)")
		return nil
	}

	fmt.Println("Replacing current executable...")
	return replaceExecutable(extractedBinaryPath, isForce)
}

func extractBinaryFromTarball(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("could not open tarball: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("could not create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return "", fmt.Errorf("error reading tar header: %w", err)
		}

		if header.Typeflag == tar.TypeReg && header.Name == repoName {
			outFile, err := os.CreateTemp("", "uptime-go-update-extracted-")
			if err != nil {
				return "", fmt.Errorf("could not create temp file for extracted binary: %w", err)
			}

			bar := progressbar.DefaultBytes(
				header.Size,
				"Extracting binary",
			)

			if _, err := io.Copy(io.MultiWriter(outFile, bar), tr); err != nil {
				outFile.Close()
				os.Remove(outFile.Name())
				return "", fmt.Errorf("could not extract binary: %w", err)
			}
			outFile.Close()

			return outFile.Name(), nil
		}
	}

	return "", errors.New("binary not found in tarball")
}

func getLatestReleaseInfo() (*gitHubRelease, error) {
	resp, err := http.Get(releaseUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status from GitHub API: %s", resp.Status)
	}

	var release gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func downloadFile(url string) (f *os.File, sha256sum string, err error) {
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("bad status: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "uptime-go-update-")
	if err != nil {
		return nil, "", err
	}

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading",
	)

	hasher := sha256.New()
	multiWriter := io.MultiWriter(tmpFile, bar, hasher)
	_, err = io.Copy(multiWriter, resp.Body)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, "", err
	}

	return tmpFile, hex.EncodeToString(hasher.Sum(nil)), nil
}

func askForConfirmation(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v. Defaulting to 'no'.\n", err)
		return false
	}

	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y"
}

// use copy instead of rename, since rename can fail across different file systems.
func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	bar := progressbar.DefaultBytes(
		sourceFileStat.Size(),
		"Copying file",
	)

	if _, err = io.Copy(io.MultiWriter(destination, bar), source); err != nil {
		return err
	}

	return destination.Sync()
}

func replaceExecutable(sourcePath string, isForce bool) error {
	exe, err := os.Executable()
	if err != nil {
		return errors.New("could not locate executable path")
	}

	exeOld := exe + ".old"
	if err := os.Rename(exe, exeOld); err != nil {
		return fmt.Errorf("failed to rename current executable: %w", err)
	}

	fmt.Println("Copying new executable...")
	if err := copyFile(sourcePath, exe); err != nil {
		if rbErr := os.Rename(exeOld, exe); rbErr != nil {
			return fmt.Errorf("failed to copy new executable to final path AND failed to roll back: %w", err)
		}
		return fmt.Errorf("failed to copy new executable: %w", err)
	}

	if err := os.Chmod(exe, 0755); err != nil {
		if rbErr := os.Rename(exeOld, exe); rbErr != nil {
			return fmt.Errorf("failed to set permissions on new executable AND failed to roll back: %w", err)
		}
		return fmt.Errorf("failed to set permissions on new executable: %w", err)
	}

	deleteOld := isForce
	if !isForce {
		if askForConfirmation("Delete old binary?") {
			deleteOld = true
		}
	}

	if deleteOld {
		if err := os.Remove(exeOld); err != nil {
			fmt.Printf("Warning: Failed to remove old binary: %v\n", err)
			fmt.Printf("You may need to remove it manually: %s\n", exeOld)
		} else {
			fmt.Println("Old binary removed.")
		}
	} else {
		fmt.Printf("Successfully updated. Old version is at %s\n", exeOld)
		fmt.Println("You can remove the .old file manually after confirming the new version works.")
	}

	return nil
}
