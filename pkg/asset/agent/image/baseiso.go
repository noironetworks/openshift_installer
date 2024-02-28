package image

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"time"

	"github.com/coreos/stream-metadata-go/arch"
	"github.com/coreos/stream-metadata-go/stream"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/agent"
	"github.com/openshift/installer/pkg/asset/agent/manifests"
	"github.com/openshift/installer/pkg/asset/agent/mirror"
	"github.com/openshift/installer/pkg/rhcos"
	"github.com/openshift/installer/pkg/types"
)

// BaseIso generates the base ISO file for the image
type BaseIso struct {
	File *asset.File
}

var (
	baseIsoFilename = ""
)

var _ asset.WritableAsset = (*BaseIso)(nil)

// Name returns the human-friendly name of the asset.
func (i *BaseIso) Name() string {
	return "BaseIso Image"
}

// getIsoFile is a pluggable function that gets the base ISO file
type getIsoFile func(archName string) (string, error)

type getIso struct {
	getter getIsoFile
}

func newGetIso(getter getIsoFile) *getIso {
	return &getIso{getter: getter}
}

// GetIsoPluggable defines the method to use get the baseIso file
var GetIsoPluggable = downloadIso

// Download the ISO using the URL in rhcos.json
func downloadIso(archName string) (string, error) {
	streamArch, err := getStreamArch(archName)
	if err != nil {
		return "", err
	}
	if artifacts, ok := streamArch.Artifacts["metal"]; ok {
		if format, ok := artifacts.Formats["iso"]; ok {
			url := format.Disk.Location

			cachedImage, err := DownloadImageFile(url)
			if err != nil {
				return "", errors.Wrapf(err, "failed to download base ISO image %s", url)
			}
			return cachedImage, nil
		}
	} else {
		return "", errors.Wrap(err, "invalid artifact")
	}

	return "", fmt.Errorf("no ISO found to download for %s", archName)
}

// Fetch RootFS URL using the rhcos.json.
func (i *BaseIso) getRootFSURL(archName string) (string, error) {
	streamArch, err := getStreamArch(archName)
	if err != nil {
		return "", err
	}
	if artifacts, ok := streamArch.Artifacts["metal"]; ok {
		if format, ok := artifacts.Formats["pxe"]; ok {
			rootFSUrl := format.Rootfs.Location
			return rootFSUrl, nil
		}
	} else {
		return "", errors.Wrap(err, "invalid artifact")
	}

	return "", fmt.Errorf("no RootFSURL found for %s", archName)
}

// Dependencies returns dependencies used by the asset.
func (i *BaseIso) Dependencies() []asset.Asset {
	return []asset.Asset{
		&manifests.AgentManifests{},
		&agent.OptionalInstallConfig{},
		&mirror.RegistriesConf{},
	}
}

func getStreamArch(archName string) (*stream.Arch, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	st, err := rhcos.FetchCoreOSBuild(ctx)
	if err != nil {
		return nil, err
	}
	streamArch, err := st.GetArchitecture(archName)
	if err != nil {
		return nil, err
	}
	return streamArch, nil
}

// Generate the baseIso
func (i *BaseIso) Generate(dependencies asset.Parents) error {

	// TODO - if image registry location is defined in InstallConfig,
	// ic := &agent.OptionalInstallConfig{}
	// p.Get(ic)

	var err error
	var baseIsoFileName string

	if urlOverride, ok := os.LookupEnv("OPENSHIFT_INSTALL_OS_IMAGE_OVERRIDE"); ok && urlOverride != "" {
		logrus.Warn("Found override for OS Image. Please be warned, this is not advised")
		baseIsoFileName, err = DownloadImageFile(urlOverride)
	} else {
		baseIsoFileName, err = i.retrieveBaseIso(dependencies)
	}

	if err == nil {
		logrus.Debugf("Using base ISO image %s", baseIsoFileName)
		i.File = &asset.File{Filename: baseIsoFileName}
		return nil
	}
	logrus.Debugf("Failed to download base ISO: %s", err)

	return errors.Wrap(err, "failed to get base ISO image")
}

func (i *BaseIso) retrieveBaseIso(dependencies asset.Parents) (string, error) {
	// use the GetIso function to get the BaseIso from the release payload
	agentManifests := &manifests.AgentManifests{}
	dependencies.Get(agentManifests)
	var baseIsoFileName string
	var err error

	// Default iso archName to x86_64.
	archName := arch.RpmArch(types.ArchitectureAMD64)

	if agentManifests.ClusterImageSet != nil {
		// If specified, use InfraEnv.Spec.CpuArchitecture for iso archName
		if agentManifests.InfraEnv.Spec.CpuArchitecture != "" {
			archName = agentManifests.InfraEnv.Spec.CpuArchitecture
		}
		releaseImage := agentManifests.ClusterImageSet.Spec.ReleaseImage
		pullSecret := agentManifests.GetPullSecretData()
		registriesConf := &mirror.RegistriesConf{}
		dependencies.Get(agentManifests, registriesConf)

		// If we have the image registry location and 'oc' command is available then get from release payload
		ocRelease := NewRelease(
			Config{MaxTries: OcDefaultTries, RetryDelay: OcDefaultRetryDelay},
			releaseImage, pullSecret, registriesConf.MirrorConfig)

		logrus.Info("Extracting base ISO from release payload")
		baseIsoFileName, err = ocRelease.GetBaseIso(archName)
		if err == nil {
			logrus.Debugf("Extracted base ISO image %s from release payload", baseIsoFileName)
			i.File = &asset.File{Filename: baseIsoFileName}
			return baseIsoFileName, nil
		}

		if errors.Is(err, fs.ErrNotExist) {
			// if image extract failed to extract the iso that architecture may be missing from release image
			return "", fmt.Errorf("base ISO for %s not found in release image, check release image architecture", archName)
		}
		if !errors.Is(err, &exec.Error{}) { // Already warned about missing oc binary
			logrus.Warning("Failed to extract base ISO from release payload - check registry configuration")
		}
	}

	logrus.Info("Downloading base ISO")
	isoGetter := newGetIso(GetIsoPluggable)
	return isoGetter.getter(archName)
}

// Files returns the files generated by the asset.
func (i *BaseIso) Files() []*asset.File {

	if i.File != nil {
		return []*asset.File{i.File}
	}
	return []*asset.File{}
}

// Load returns the cached baseIso
func (i *BaseIso) Load(f asset.FileFetcher) (bool, error) {

	if baseIsoFilename == "" {
		return false, nil
	}

	baseIso, err := f.FetchByName(baseIsoFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrap(err, fmt.Sprintf("failed to load %s file", baseIsoFilename))
	}

	i.File = baseIso
	return true, nil
}
