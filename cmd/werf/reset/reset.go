package reset

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flant/werf/cmd/werf/common"
	"github.com/flant/werf/cmd/werf/common/docker_authorizer"
	"github.com/flant/werf/pkg/cleanup"
	"github.com/flant/werf/pkg/docker"
	"github.com/flant/werf/pkg/lock"
	"github.com/flant/werf/pkg/werf"
)

var CmdData struct {
	OnlyCacheVersion bool

	DryRun bool
}

var CommonCmdData common.CmdData

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "reset",
		DisableFlagsInUseLine: true,
		Short: "Delete all images, containers, and cache files for all projects created by werf on the host",
		Long: common.GetLongCommandDescription(`Delete all images, containers, and cache files for all projects created by werf on the host.

Reset is the fullest method of cleaning on the local machine.

No project files (i.e. werf.yaml) are needed to run reset.

See more info about reset type of cleaning: https://flant.github.io/werf/reference/registry/cleaning.html#reset`),
		RunE: func(cmd *cobra.Command, args []string) error {
			common.LogVersion()

			err := runReset()
			if err != nil {
				return fmt.Errorf("reset failed: %s", err)
			}

			return nil
		},
	}

	common.SetupTmpDir(&CommonCmdData, cmd)
	common.SetupHomeDir(&CommonCmdData, cmd)

	//cmd.Flags().BoolVarP(&CmdData.OnlyDevModeCache, "only-dev-mode-cache", "", false, "delete stages cache, images, and containers created in developer mode")
	cmd.Flags().BoolVarP(&CmdData.OnlyCacheVersion, "only-cache-version", "", false, "Only delete stages cache, images, and containers created by these werf versions which are incompatible with current werf version")

	cmd.Flags().BoolVarP(&CmdData.DryRun, "dry-run", "", false, "Indicate what the command would do without actually doing that")

	return cmd
}

func runReset() error {
	if err := werf.Init(*CommonCmdData.TmpDir, *CommonCmdData.HomeDir); err != nil {
		return fmt.Errorf("initialization error: %s", err)
	}

	if err := lock.Init(); err != nil {
		return err
	}

	if err := docker.Init(docker_authorizer.GetHomeDockerConfigDir()); err != nil {
		return err
	}

	commonOptions := cleanup.CommonOptions{DryRun: CmdData.DryRun}
	if CmdData.OnlyCacheVersion {
		return cleanup.ResetCacheVersion(commonOptions)
	} else {
		return cleanup.ResetAll(commonOptions)
	}

	return nil
}
