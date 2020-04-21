package manifests

import (
	"archive/tar"
        "bytes"
	"compress/gzip"
	"fmt"
	yamlv2 "gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/templates/content/openshift"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	noCrdFilename = filepath.Join(manifestDir, "cluster-network-01-crd.yml")
	noCfgFilename = filepath.Join(manifestDir, "cluster-network-02-config.yml")
	noSNATCRFilename = filepath.Join(manifestDir, "cluster-network-27-snat-policy-cr.yaml")
)

var snatCRTmpl = template.Must(template.New("snat-cr").Parse(`apiVersion: aci.snat/v1
kind: SnatPolicy
metadata:
  name: installerclusterdefault
spec:
  snatIp:
    -  {{.snatIP}}
`))

var rdConfigTmpl = template.Must(template.New("rdconfig").Parse(`apiVersion: aci.snat/v1
kind: RdConfig
metadata:
  name: routingdomain-config
  namespace: aci-containers-system
spec:
  usersubnets:
  - {{ .neutronCIDR }}
  - 224.0.0.0/4
`))

var clusterNetwork03Tmpl = template.Must(template.New("cluster03").Parse(`apiVersion: operator.openshift.io/v1
kind: Network
metadata:
  name: cluster
spec:
  disableMultiNetwork: true
  clusterNetwork:
  - cidr: {{.clusterNet}}
    hostPrefix: {{.hostPrefix}}
  defaultNetwork:
    type: {{.netType}}
  serviceNetwork:
  - {{.svcNet}}
`))

// We need to manually create our CRDs first, so we can create the
// configuration instance of it in the installer. Other operators have
// their CRD created by the CVO, but we need to create the corresponding
// CRs in the installer, so we need the CRD to be there.
// The first CRD is the high-level Network.config.openshift.io object,
// which is stable and minimal. Administrators can configure the
// network in a more detailed manner with the operator-specific CR, which
// also needs to be done before the installer is run, so we provide both.

// Networking generates the cluster-network-*.yml files.
type Networking struct {
	Config   *configv1.Network
	FileList []*asset.File
}

var _ asset.WritableAsset = (*Networking)(nil)

type AciContainersConfig struct {
        Data ConfigData `yaml:data,omitempty`
}

type ConfigData struct {
        HostConfig string `yaml:"host-agent-config"`
}

type HostConfigMap struct {
        ServiceVLAN int    `yaml:"service-vlan"`
        InfraVLAN   int    `yaml:"aci-infra-vlan"`
        KubeApiVLAN int    `yaml:"kubeapi-vlan"`
        PodSubnet   string `yaml:"pod-subnet"`
        NodeSubnet  string `yaml:"node-subnet"`
}

type ClusterConfig03 struct {
	ApiVersion string     `yaml:"apiVersion"`
        Kind       string     `yaml:"kind"`
        Metadata   MetaEntry  `yaml:"metadata,omitempty"`
	Spec       SpecEntry  `yaml:"spec,omitempty"`
}

type MetaEntry struct {
	Name	string `yaml:"name"`
}

type SpecEntry struct {
	Multus		bool				`yaml:"disableMultiNetwork"`
        ClusterNetwork	[]ClusterEntry 			`yaml:"clusterNetwork,omitempty"`  
        DefaultNetwork  DefaultNetEntry			`yaml:"defaultNetwork,omitempty"`
        NetworkType	string				`yaml:"networkType,omitempty"`
        ServiceNetwork	[]string			`yaml:"serviceNetwork,omitempty"`
}

type ClusterEntry struct {
        CIDR		string	`yaml:"cidr"`
	HostPrefix	int32	`yaml:"hostPrefix"`
}

type DefaultNetEntry struct {
	Type	string	`yaml:"type"`
}

// Name returns a human friendly name for the operator.
func (no *Networking) Name() string {
	return "Network Config"
}

// Dependencies returns all of the dependencies directly needed to generate
// network configuration.
func (no *Networking) Dependencies() []asset.Asset {
	return []asset.Asset{
		&installconfig.InstallConfig{},
		&openshift.NetworkCRDs{},
	}
}

// Generate generates the network operator config and its CRD and the SNAT Cluster CR if needed.
func (no *Networking) Generate(dependencies asset.Parents) error {
	installConfig := &installconfig.InstallConfig{}
	crds := &openshift.NetworkCRDs{}
	dependencies.Get(installConfig, crds)

	netConfig := installConfig.Config.Networking

	clusterNet := []configv1.ClusterNetworkEntry{}
	if len(netConfig.ClusterNetwork) > 0 {
		for _, net := range netConfig.ClusterNetwork {
			clusterNet = append(clusterNet, configv1.ClusterNetworkEntry{
				CIDR:       net.CIDR.String(),
				HostPrefix: uint32(net.HostPrefix),
			})
		}
	} else {
		return errors.Errorf("ClusterNetworks must be specified")
	}

	serviceNet := []string{}
	for _, sn := range netConfig.ServiceNetwork {
		serviceNet = append(serviceNet, sn.String())
	}

	no.Config = &configv1.Network{
		TypeMeta: metav1.TypeMeta{
			APIVersion: configv1.SchemeGroupVersion.String(),
			Kind:       "Network",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
			// not namespaced
		},
		Spec: configv1.NetworkSpec{
			ClusterNetwork: clusterNet,
			ServiceNetwork: serviceNet,
			NetworkType:    netConfig.NetworkType,
			// Block all Service.ExternalIPs by default
			ExternalIP: &configv1.ExternalIPConfig{
				Policy: &configv1.ExternalIPPolicy{},
			},
		},
		Status: configv1.NetworkStatus{
			ClusterNetwork: clusterNet,
			ServiceNetwork: serviceNet,
			NetworkType:    netConfig.NetworkType,
		},
	}

	configData, err := yaml.Marshal(no.Config)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s manifests from InstallConfig", no.Name())
	}

	crdContents := ""
	for _, crdFile := range crds.Files() {
		crdContents = fmt.Sprintf("%s\n---\n%s", crdContents, crdFile.Data)
	}

	no.FileList = []*asset.File{
                {
                        Filename: noCrdFilename,
                        Data:     []byte(crdContents),
                },
                {
                        Filename: noCfgFilename,
                        Data:     configData,
                },
        }

	err = CiscoAciValidation(installConfig)
	if err != nil {
		return err
	}

	err = no.CiscoAciManifest(installConfig, netConfig)
	if err != nil {
		return err
	}

	return nil
}

func CiscoAciValidation(installConfig *installconfig.InstallConfig) error {

	// Check if installConfg input has a valid installerHostSubnet value
	_, err := ipnet.ParseCIDR(installConfig.Config.Platform.OpenStack.AciNetExt.InstallerHostSubnet)
	if err != nil {
		return errors.WithMessage(err, "Please use valid value of installerHostSubnet")
	}

	r, err := os.Open(installConfig.Config.Platform.OpenStack.AciNetExt.ProvisionTar)
        if err != nil {
		return errors.WithMessage(err, "Invalid provisionTar path/file")
	}
	config, err := ExtractTarGz(r)
	if err != nil {
		return errors.WithMessage(err, "Unable to extract yamls from provisionTar")
	}
	installConfig.Config.Platform.OpenStack.AciNetExt.KubeApiVLAN = strconv.Itoa(config.KubeApiVLAN)
	installConfig.Config.Platform.OpenStack.AciNetExt.InfraVLAN = strconv.Itoa(config.InfraVLAN)
	installConfig.Config.Platform.OpenStack.AciNetExt.ServiceVLAN = strconv.Itoa(config.ServiceVLAN)
	machineCIDR := &installConfig.Config.Networking.MachineNetwork[0].CIDR
	clusterNetworkCIDR := &installConfig.Config.Networking.ClusterNetwork[0].CIDR
	nodeDiff := DiffSubnets(config.NodeSubnet, machineCIDR)
    	if nodeDiff != nil {
		option := UserPrompt(nodeDiff.String(), machineCIDR, "node_subnet", "machineCIDR")
		if (option == true) {
			installConfig.Config.Networking.DeprecatedMachineCIDR, _ = ipnet.ParseCIDR(nodeDiff.String())
			log.Print("Setting machineCIDR to " + nodeDiff.String())
		} else {
			err := errors.New("node_subnet in acc-provision input(" + nodeDiff.String() + ") has to be the same as machineCIDR in install-config.yaml(" + machineCIDR.String() + ")") 
			return err
		}
    	}
	clusterDiff := DiffSubnets(config.PodSubnet, clusterNetworkCIDR)
    	if clusterDiff != nil {
		option := UserPrompt(clusterDiff.String(), clusterNetworkCIDR, "pod_subnet", "clusterNetworkCIDR")
		if (option == true) {
			parsedCIDR, _ := ipnet.ParseCIDR(clusterDiff.String())
			installConfig.Config.Networking.ClusterNetwork[0].CIDR = *parsedCIDR
			log.Print("Setting clusterNetwork CIDR to " + clusterDiff.String())
		} else {
			err := errors.New("pod_subnet in acc-provision input(" + clusterDiff.String() + ") has to be the same as clusterNetwork:cidr in install-config.yaml(" + clusterNetworkCIDR.String() + ")")
			return err
		}
    	}

	return nil
}

func DiffSubnets(sub1 string, sub2 *ipnet.IPNet) *net.IPNet {
        // Returns first subnet if the subnets are different
        _, net1, _ := net.ParseCIDR(sub1)
        if net1.String() != sub2.String() {
                return net1
	}
        return nil
}

func UserPrompt(sub1 string, sub2 *ipnet.IPNet, item1 string, item2 string) bool {
	var option string
	log.Print("There's a discrepancy between " + item1 + "(" + sub1 + ") in acc-provision input and " + item2 + "(" + sub2.String() + ") in install-config.yaml")
	log.Print("Enter Y to use acc-provision value, or N to exit installer and fix acc-provision tar")
	fmt.Scanln(&option)
	var op bool
	if (option == "y" || option == "Y") {
		op = true
	}
	return op
}

func ExtractTarGz(gzipStream io.Reader) (HostConfigMap, error) {
	config := HostConfigMap{}
        uncompressedStream, err := gzip.NewReader(gzipStream)
        if err != nil {
		return config, err
        }

        tarReader := tar.NewReader(uncompressedStream)

        for true {
                header, err := tarReader.Next()

                if err == io.EOF {
                        break
                }

                if err != nil {
			return config, err
                }

                switch header.Typeflag {
                case tar.TypeReg:
                        temp, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return config, err
			}

			// Unmarshal acc configmap to get acc-provision values
                        if strings.Contains(header.Name, "aci-containers-config") {
                                t := AciContainersConfig{}
                                err = yamlv2.Unmarshal(temp, &t)
                                if err != nil {
					return config, err
                                }
                                err = yamlv2.Unmarshal([]byte(t.Data.HostConfig), &config)
                                if err != nil {
					return config, err
                                }
                        }
                default:
			return config, errors.New("Unsupported file type in tar")
		}

        }
        return config, nil
}

func (no *Networking) CiscoAciManifest(installConfig *installconfig.InstallConfig, netConfig *types.Networking) error {
	// Untar and add acc-provision files
	r, _ := os.Open(installConfig.Config.Platform.OpenStack.AciNetExt.ProvisionTar)
	uncompressedStream, _ := gzip.NewReader(r)
	tarReader := tar.NewReader(uncompressedStream)
	var noRDconfigFilename string
	for true {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		b, _ := ioutil.ReadAll(tarReader);

		// Save filenumber of hostagent daemonset for rdconfig CR
		if strings.Contains(header.Name, "DaemonSet-aci-containers-host"){
			hostAgentFileName := header.Name
			hyphenParsed := strings.Split(hostAgentFileName, "-")
			hostAgentFileNo, err := strconv.Atoi(hyphenParsed[2])
			if err != nil {
				return errors.Wrapf(err, "failed to decipher host agent  manifest file from acc-provision tar")
			}
			rdConfigFileNo := strconv.Itoa(hostAgentFileNo - 4)
			noRDconfigFilename = filepath.Join(manifestDir, "cluster-network-" + rdConfigFileNo + "-2-CustomResource-rdconfig.yaml")
		}

		// Edit cluster-network-03 file with correct fields
		if strings.Contains(header.Name, "cluster-network-03"){
			cluster03Data := &bytes.Buffer{}
			clusterNetworkCIDR := &netConfig.ClusterNetwork[0].CIDR
			data := map[string]string{"clusterNet": clusterNetworkCIDR.String(),
                		"hostPrefix":  strconv.Itoa(int(netConfig.ClusterNetwork[0].HostPrefix)),
                		"netType": netConfig.NetworkType, "svcNet": netConfig.ServiceNetwork[0].String()}
			if err := clusterNetwork03Tmpl.Execute(cluster03Data, data); err != nil {
				return errors.Wrapf(err, "failed to create cluster-network-03 manifests from InstallConfig")
			}
			b = cluster03Data.Bytes()

		}
		tempFile := &asset.File{Filename: filepath.Join(manifestDir, header.Name), Data: b}
		no.FileList = append(no.FileList, tempFile)
	}

	// Create SNAT Cluster CR file 
	if installConfig.Config.Platform.OpenStack.AciNetExt.ClusterSNATSubnet != "" {
		snatData := &bytes.Buffer{}
		data := map[string]string{"snatIP": installConfig.Config.Platform.OpenStack.AciNetExt.ClusterSNATSubnet}
		if err := snatCRTmpl.Execute(snatData, data); err != nil {
			return errors.Wrapf(err, "failed to create SNAT CR manifests from InstallConfig")
		}
		// add destIP if field present
		if installConfig.Config.Platform.OpenStack.AciNetExt.ClusterSNATDest != "" {
			dest := "  destIp:\n    -  " + installConfig.Config.Platform.OpenStack.AciNetExt.ClusterSNATDest + "\n"
			snatData.WriteString(dest)
		}
		snatFile := &asset.File{Filename: noSNATCRFilename, Data: snatData.Bytes()}
		no.FileList = append(no.FileList, snatFile)

		// Create yaml for rdConfig
		if noRDconfigFilename == "" {
			return errors.New("no manifest with DaemonSet-aci-containers-host found in acc-provision tar")
		}
                rdConfigData := &bytes.Buffer{}
                data = map[string]string{"neutronCIDR": installConfig.Config.Platform.OpenStack.AciNetExt.NeutronCIDR.String()}
                if err := rdConfigTmpl.Execute(rdConfigData, data); err != nil {
                        return errors.Wrapf(err, "failed to create rdconfig manifest from InstallConfig")
                }
                rdconfigFile := &asset.File{Filename: noRDconfigFilename, Data: rdConfigData.Bytes()}
                no.FileList = append(no.FileList, rdconfigFile)
	}

	return nil

}

// Files returns the files generated by the asset.
func (no *Networking) Files() []*asset.File {
	return no.FileList
}

// Load returns false since this asset is not written to disk by the installer.
func (no *Networking) Load(f asset.FileFetcher) (bool, error) {
	return false, nil
}
