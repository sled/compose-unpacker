package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/portainer/docker-compose-wrapper/compose"
	"github.com/portainer/portainer/api/filesystem"
)

var errDeployComposeFailure = errors.New("compose stack deployment failure")

func (cmd *DeployCommand) Run(cmdCtx *CommandExecutionContext) error {
	cmdCtx.logger.Infow("Deploying Compose stack from Git repository",
		"repository", cmd.GitRepository,
		"composePath", cmd.ComposeRelativeFilePaths,
		"destination", cmd.Destination,
	)

	if cmd.User != "" && cmd.Password != "" {
		cmdCtx.logger.Infow("Using Git authentication",
			"user", cmd.User,
			"password", "<redacted>",
		)
	}

	i := strings.LastIndex(cmd.GitRepository, "/")
	if i == -1 {
		cmdCtx.logger.Errorw("Invalid Git repository URL",
			"repository", cmd.GitRepository,
		)

		return errDeployComposeFailure
	}
	repositoryName := strings.TrimSuffix(cmd.GitRepository[i+1:], ".git")

	cmdCtx.logger.Infow("Checking the file system...",
		"directory", cmd.Destination,
	)
	if _, err := os.Stat(cmd.Destination); err != nil {
		if os.IsNotExist(err) {
			cmdCtx.logger.Infow("Creating folder in the file system...",
				"directory", cmd.Destination,
			)
			err := os.MkdirAll(cmd.Destination, 0755)
			if err != nil {
				cmdCtx.logger.Errorw("Failed to create destination directory",
					"error", err,
				)
				return errDeployComposeFailure
			}
		} else {
			return err
		}
	} else {
		cmdCtx.logger.Infow("Backing up folder in the file system...",
			"directory", cmd.Destination,
		)
		backupProjectPath := fmt.Sprintf("%s-old", cmd.Destination)
		err = filesystem.MoveDirectory(cmd.Destination, backupProjectPath)
		if err != nil {
			return err
		}
		defer func() {
			err = os.RemoveAll(backupProjectPath)
			if err != nil {
				log.Printf("[WARN] [http,stacks,git] [error: %s] [message: unable to remove git repository directory]", err)
			}
		}()
	}
	cmdCtx.logger.Infow("Creating target destination directory on disk",
		"directory", cmd.Destination,
	)
	gitOptions := git.CloneOptions{
		URL:   cmd.GitRepository,
		Auth:  getAuth(cmd.User, cmd.Password),
		Depth: 1,
	}

	clonePath := path.Join(cmd.Destination, repositoryName)

	cmdCtx.logger.Infow("Cloning git repository",
		"path", clonePath,
		"cloneOptions", gitOptions,
	)

	_, err := git.PlainCloneContext(cmdCtx.context, clonePath, false, &gitOptions)
	if err != nil {
		cmdCtx.logger.Errorw("Failed to clone Git repository",
			"error", err,
		)

		return errDeployComposeFailure
	}

	cmdCtx.logger.Infow("Creating Compose deployer",
		"binPath", BIN_PATH,
	)

	deployer, err := compose.NewComposeDeployer(BIN_PATH, "")
	if err != nil {
		cmdCtx.logger.Errorw("Failed to create Compose deployer",
			"error", err,
		)

		return errDeployComposeFailure
	}
	composeFilePaths := make([]string, len(cmd.ComposeRelativeFilePaths))
	for i := 0; i < len(cmd.ComposeRelativeFilePaths); i++ {
		composeFilePaths[i] = path.Join(clonePath, cmd.ComposeRelativeFilePaths[i])
	}

	cmdCtx.logger.Infow("Deploying Compose stack",
		"composeFilePaths", composeFilePaths,
		"workingDirectory", clonePath,
		"projectName", cmd.ProjectName,
	)

	err = deployer.Deploy(cmdCtx.context, clonePath, "", cmd.ProjectName, composeFilePaths, "", false)
	if err != nil {
		cmdCtx.logger.Errorw("Failed to deploy Compose stack",
			"error", err,
		)

		return errDeployComposeFailure
	}

	cmdCtx.logger.Info("Compose stack deployment complete")

	return nil
}

func getAuth(username, password string) *http.BasicAuth {
	if password != "" {
		if username == "" {
			username = "token"
		}

		return &http.BasicAuth{
			Username: username,
			Password: password,
		}
	}
	return nil
}
