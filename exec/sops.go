package exec

import (
	"io/fs"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

func DecryptSops(composeFilePaths []string, env []string) error {
	// get the root folders of the compose files
	composeRootFolderPaths := findRootPaths(composeFilePaths)

	var sopsFilePaths []string
	for _, rootFolderPath := range composeRootFolderPaths {
		log.Info().
			Str("path", rootFolderPath).
			Msg("Walking directory, looking for SOPS files...")

		filepath.WalkDir(rootFolderPath, func(path string, file fs.DirEntry, err error) error {
			if err != nil {
				log.Warn().
					Err(err).
					Str("path", path).
					Msg("Encountered an error while collecting SOPS file paths")
				return nil
			}

			if !file.IsDir() {
				matched, err := filepath.Match("*.sops.*", file.Name())
				if err != nil {
					return err
				}

				if matched {
					log.Info().
						Str("path", path).
						Msg("Found a SOPS file")

					sopsFilePaths = append(sopsFilePaths, path)
				}
			}
			return nil
		})
	}

	command := getSopsBinaryPath()

	for _, sopsFilePath := range sopsFilePaths {
		sopsFileFolderPath := filepath.Dir(sopsFilePath)
		sopsFileName := filepath.Base(sopsFilePath)
		outputFileName := strings.ReplaceAll(sopsFileName, ".sops.", ".")

		args := make([]string, 0)
		args = append(args, "--output", outputFileName, "--decrypt", sopsFileName)

		err := RunCommandAndCaptureStdErr(command, args, env, sopsFileFolderPath)
		if err != nil {
			log.Warn().
				Str("command", command).
				Str("env", strings.Join(env, "; ")).
				Str("workingDir", sopsFileFolderPath).
				Str("args", strings.Join(args, " ")).
				Err(err).
				Msg("Failed to decrypt SOPS file")

			// TODO: Should we continue instead of aborting the deployment?
			return err
		}
	}

	return nil
}

func getSopsBinaryPath() string {
	command := path.Join(BIN_PATH, "sops")
	if runtime.GOOS == "windows" {
		command = path.Join(BIN_PATH, "sops.exe")
	}
	return command
}

func findRootPaths(filePaths []string) []string {
	if len(filePaths) == 0 {
		return []string{}
	}

	// remove filename from input paths so we're left with folder paths
	folderPaths := make([]string, len(filePaths))
	for i := 0; i < len(filePaths); i++ {
		folderPaths[i] = filepath.Clean(filepath.Dir(filePaths[i]))
	}

	// sort, so that common paths are adjacent
	/*
		/home/stacks/a
		/home/stacks/a/nested
		/home/stecks/a/nested/another
		/home/stacks/b
		/home/stacks/b/nested
		....
	*/
	sort.Strings(folderPaths)

	// because we encounter the root path first, ignore all consecutive
	// paths that start with the previous folder path.
	rootPaths := []string{folderPaths[0]}
	for i := 0; i < len(folderPaths); i++ {
		if !strings.HasPrefix(folderPaths[i], rootPaths[len(rootPaths)-1]) {
			rootPaths = append(rootPaths, folderPaths[i])
		}
	}

	return rootPaths
}
