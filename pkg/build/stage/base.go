package stage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flant/dapp/pkg/config"
	"github.com/flant/dapp/pkg/image"
	"github.com/flant/dapp/pkg/slug"
	"github.com/flant/dapp/pkg/util"
)

type StageName string

const (
	From                        StageName = "from"
	BeforeInstall               StageName = "before_install"
	ArtifactImportBeforeInstall StageName = "before_install_artifact"
	GAArchive                   StageName = "g_a_archive"
	GAPreInstallPatch           StageName = "g_a_pre_install_patch"
	Install                     StageName = "install"
	ArtifactImportAfterInstall  StageName = "after_install_artifact"
	GAPostInstallPatch          StageName = "g_a_post_install_patch"
	BeforeSetup                 StageName = "before_setup"
	ArtifactImportBeforeSetup   StageName = "before_setup_artifact"
	GAPreSetupPatch             StageName = "g_a_pre_setup_patch"
	Setup                       StageName = "setup"
	ArtifactImportAfterSetup    StageName = "after_setup_artifact"
	GAPostSetupPatch            StageName = "g_a_post_setup_patch"
	GALatestPatch               StageName = "g_a_latest_patch"
	DockerInstructions          StageName = "docker_instructions"
	GAArtifactPatch             StageName = "g_a_artifact_patch"
	BuildArtifact               StageName = "build_artifact"
)

type NewBaseStageOptions struct {
	DimgTmpDir          string
	DimgContainerTmpDir string
	ProjectBuildDir     string
}

func newBaseStage(options *NewBaseStageOptions) *BaseStage {
	s := &BaseStage{}
	s.projectBuildDir = options.ProjectBuildDir
	s.dimgTmpDir = options.DimgTmpDir
	s.dimgContainerTmpDir = options.DimgContainerTmpDir
	return &BaseStage{}
}

type BaseStage struct {
	signature           string
	image               image.Image
	gitArtifacts        []*GitArtifact
	dimgConfig          *config.Dimg
	dimgTmpDir          string
	dimgContainerTmpDir string
	projectBuildDir     string
}

func (s *BaseStage) Name() StageName {
	panic("method must be implemented!")
}

func (s *BaseStage) GetDependencies(_ Conveyor, _ image.Image) (string, error) {
	panic("method must be implemented!")
}

func (s *BaseStage) IsEmpty(_ Conveyor, _ image.Image) (bool, error) {
	panic("method must be implemented!")
}

func (s *BaseStage) GetContext(_ Conveyor) (string, error) {
	return "", nil
}

func (s *BaseStage) GetRelatedStageName() StageName {
	return ""
}

func (s *BaseStage) PrepareImage(_ Conveyor, prevBuiltImage, image image.Image) error {
	var err error

	/*
	 * NOTE: BaseStage.PrepareImage does not called in From.PrepareImage.
	 * NOTE: Take into account when adding new base PrepareImage steps.
	 */

	err = s.addServiceMounts(prevBuiltImage, image, false)
	if err != nil {
		return fmt.Errorf("error adding service mounts: %s", err)
	}

	err = s.addCustomMounts(prevBuiltImage, image, false)
	if err != nil {
		return fmt.Errorf("error adding custom mounts: %s", err)
	}

	return nil
}

func (s *BaseStage) addServiceMounts(prevBuiltImage, image image.Image, onlyLabels bool) error {
	mountpointsByType := map[string][]string{}

	for _, mountCfg := range s.dimgConfig.Mount {
		mountpoint := filepath.Clean(mountCfg.To)
		mountpointsByType[mountCfg.Type] = append(mountpointsByType[mountCfg.Type], mountpoint)
	}

	var labels map[string]string
	if prevBuiltImage != nil {
		labels = prevBuiltImage.Labels()
	}

	for _, labelMountType := range []struct{ Label, MountType string }{
		struct{ Label, MountType string }{"dapp-mount-tmp-dir", "tmp_dir"},
		struct{ Label, MountType string }{"dapp-mount-build-dir", "build_dir"},
	} {
		v, hasKey := labels[labelMountType.Label]
		if !hasKey {
			continue
		}

		mountpoints := util.RejectEmptyStrings(util.UniqStrings(strings.Split(v, ";")))
		mountpointsByType[labelMountType.MountType] = mountpoints
	}

	for mountType, mountpoints := range mountpointsByType {
		if !onlyLabels {
			for _, mountpoint := range mountpoints {
				absoluteMountpoint := filepath.Join("/", mountpoint)

				var absoluteFrom string
				switch mountType {
				case "tmp_dir":
					absoluteFrom = filepath.Join(s.dimgTmpDir, "mount", slug.Slug(absoluteMountpoint))
				case "build_dir":
					absoluteFrom = filepath.Join(s.projectBuildDir, "mount", slug.Slug(absoluteMountpoint))
				default:
					panic(fmt.Sprintf("unknown mount type %s", mountType))
				}

				err := os.MkdirAll(absoluteFrom, os.ModePerm)
				if err != nil {
					return fmt.Errorf("error creating tmp path %s for mount: %s", absoluteFrom, err)
				}

				image.Container().RunOptions().AddVolume(fmt.Sprintf("%s:%s", absoluteFrom, absoluteMountpoint))
			}
		}

		var labelName string
		switch mountType {
		case "tmp_dir":
			labelName = "dapp-mount-type-tmp-dir"
		case "build_dir":
			labelName = "dapp-mount-type-build-dir"
		default:
			panic(fmt.Sprintf("unknown mount type %s", mountType))
		}

		labelValue := strings.Join(mountpoints, ";")

		image.Container().ServiceCommitChangeOptions().AddLabel(map[string]string{labelName: labelValue})
	}

	return nil
}

func (s *BaseStage) addCustomMounts(prevBuiltImage, image image.Image, onlyLabels bool) error {
	mountpointsByFrom := map[string][]string{}

	for _, mountCfg := range s.dimgConfig.Mount {
		if mountCfg.Type != "custom_dir" {
			continue
		}

		from := filepath.Clean(mountCfg.From)
		mountpoint := filepath.Clean(mountCfg.To)

		mountpointsByFrom[from] = util.UniqAppendString(mountpointsByFrom[from], mountpoint)
	}

	var labels map[string]string
	if prevBuiltImage != nil {
		labels = prevBuiltImage.Labels()
	}

	for k, v := range labels {
		if !strings.HasPrefix(k, "dapp-mount-custom-dir-") {
			continue
		}

		parts := strings.SplitN(k, "dapp-mount-custom-dir-", 2)
		from := strings.Replace(parts[1], "--", "/", -1)

		mountpoints := util.RejectEmptyStrings(util.UniqStrings(strings.Split(v, ";")))
		mountpointsByFrom[from] = mountpoints
	}

	for from, mountpoints := range mountpointsByFrom {
		absoluteFrom := util.ExpandPath(from)

		err := os.MkdirAll(absoluteFrom, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating %s: %s", absoluteFrom, err)
		}

		if !onlyLabels {
			for _, mountpoint := range mountpoints {
				absoluteMountpoint := filepath.Join("/", mountpoint)
				image.Container().RunOptions().AddVolume(fmt.Sprintf("%s:%s", absoluteFrom, absoluteMountpoint))
			}
		}

		labelName := fmt.Sprintf("dapp-mount-custom-dir-%s", strings.Replace(from, "/", "--", -1))
		labelValue := strings.Join(mountpoints, ";")
		image.Container().ServiceCommitChangeOptions().AddLabel(map[string]string{labelName: labelValue})
	}

	return nil
}

func (s *BaseStage) SetSignature(signature string) {
	s.signature = signature
}

func (s *BaseStage) GetSignature() string {
	return s.signature
}

func (s *BaseStage) SetImage(image image.Image) {
	s.image = image
}

func (s *BaseStage) GetImage() image.Image {
	return s.image
}

func (s *BaseStage) SetGitArtifacts(gitArtifacts []*GitArtifact) {
	s.gitArtifacts = gitArtifacts
}

func (s *BaseStage) GetGitArtifacts() []*GitArtifact {
	return s.gitArtifacts
}
