package deployer

import (
	"crypto/rand"
	_ "embed"
	"fmt"
	"golden/pkg/fsys"
	"golden/pkg/inventory"
	"golden/pkg/manifest"
	"golden/pkg/rerrors"
	"golden/pkg/rtemplate"
	"golden/pkg/sh"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type Deployer struct {
	resolvedInstanceVars map[string]map[string]interface{}
	inv                  *inventory.Inventory
	report               *Report
	hosts                []string
	hostToInstances      map[string][]*inventory.Instance
	parsedTemplates      map[string]*template.Template
	sshControlPath       string
	localTmpDir          string
	remoteTmpDir         string
}

func New(resolvedInstanceVars map[string]map[string]interface{}, inv *inventory.Inventory) *Deployer {
	return &Deployer{
		resolvedInstanceVars: resolvedInstanceVars,
		inv:                  inv,
		report:               NewReport(),
		hosts:                []string{},
		hostToInstances:      map[string][]*inventory.Instance{},
		parsedTemplates:      map[string]*template.Template{},
		sshControlPath:       "",
		localTmpDir:          "",
		remoteTmpDir:         "",
	}
}

func (d *Deployer) Deploy(manif manifest.Manifest, appsWhitelist []string) *Report {
	appsWhiteMap := map[string]struct{}{}
	for _, app := range appsWhitelist {
		appsWhiteMap[app] = struct{}{}
	}

	d.constructTmpDirNames()
	for _, inst := range d.inv.GetInstancesForManifest(manif) {
		if len(appsWhiteMap) > 0 {
			if _, ok := appsWhiteMap[inst.App]; !ok {
				continue
			}
		}
		host := inst.Host
		if list, ok := d.hostToInstances[host]; ok {
			d.hostToInstances[host] = append(list, inst)
			continue
		}
		list := make([]*inventory.Instance, 1)
		list[0] = inst
		d.hosts = append(d.hosts, host)
		d.hostToInstances[host] = list
	}

	if len(d.hosts) == 0 {
		return d.report
	}

	err := os.Mkdir(d.localTmpDir, 0744)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(d.localTmpDir)

	for _, h := range d.hosts {
		d.deployToHost(h)
	}

	return d.report
}

func (d *Deployer) constructTmpDirNames() {
	pid := os.Getpid()
	date := time.Now().UTC().Format("20060102")
	random, err := rand.Int(rand.Reader, big.NewInt(1024))
	if err != nil {
		panic(err)
	}
	d.localTmpDir = fmt.Sprintf(".golden-local-%s-%d-%x", date, pid, random)
	d.remoteTmpDir = fmt.Sprintf(".golden-remote-%s-%d-%x", date, pid, random)
	d.sshControlPath = filepath.Join(d.localTmpDir, ".golden-ssh-control-path")
}

func (d *Deployer) deployToHost(host string) {
	hostData := d.inv.GetHost(host)
	fmt.Fprintf(os.Stderr, "==> Processing instances for %s %s <==\n", host, hostData)

	err := os.Mkdir(filepath.Join(d.localTmpDir, host), 0755)
	if err != nil {
		panic(err)
	}

	r := d.report.CreateHostReport(host)
	for _, inst := range d.hostToInstances[host] {
		d.packInstance(inst, r)
	}

	r.HostPackingStarted()
	packedHostPath := filepath.Join(d.localTmpDir, host) + ".tar.gz"
	sh.MustDoSilentlyf("tar -C %s -cvzf %s %s", d.localTmpDir, packedHostPath, host)
	r.HostPackingDone()

	if err := os.RemoveAll(filepath.Join(d.localTmpDir, host)); err != nil {
		panic(err)
	}

	r.DeployStarted()
	defer func() {
		recov := recover()
		if recov != nil {
			r.InstanceDeployDone(false)
			r.DeployDone()
			panic(recov)
		}
		r.DeployDone()
	}()
	installPrefixRoot := ""
	var executor sh.Executor
	hostRemoteTmpDir := d.remoteTmpDir
	if !hostData.IsLocalHost() {
		ssh := sh.NewSshSession(d.sshControlPath, hostData.GetSshConnStr())
		defer ssh.Close()
		executor = ssh
	} else if !hostData.IsThisUser() {
		executor = sh.NewSudo(hostData.GetUser())
	} else {
		executor = sh.Shell
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		hostRemoteTmpDir = filepath.Join(homeDir, d.remoteTmpDir)
		installPrefixRoot = homeDir
	}
	executor.MustDoSilentlyf("mkdir %s", hostRemoteTmpDir)
	defer executor.MustDoSilentlyf("rm -rf %s", hostRemoteTmpDir)
	fmt.Fprintf(os.Stderr, "%s Transferring an archive for %s\n", hostData, host)
	executor.MustCp(packedHostPath, hostRemoteTmpDir)
	executor.MustDoSilentlyf(
		"tar --no-same-owner -C %s -xvzf %s.tar.gz",
		hostRemoteTmpDir,
		filepath.Join(hostRemoteTmpDir, host),
	)
	executor.MustDoSilentlyf("rm -rf %s.tar.gz", filepath.Join(hostRemoteTmpDir, host))
	for _, inst := range d.hostToInstances[host] {
		r.InstanceDeployStarted()
		var deployPath string
		if filepath.IsAbs(inst.InstallPrefix) || strings.HasPrefix(inst.InstallPrefix, "~") {
			deployPath = inst.InstallPrefix
		} else {
			deployPath = filepath.Join(installPrefixRoot, inst.InstallPrefix)
		}
		if strings.TrimSpace(deployPath) == "" {
			deployPath = "."
		}
		fmt.Fprintf(os.Stderr, "%s unpacking %s to %s\n", hostData, inst.Name, deployPath)
		executor.MustDoSilentlyf("mkdir -p %s", deployPath)
		executor.MustDoSilentlyf(
			"tar --no-same-owner -C %s -xvf %s.tar",
			deployPath,
			filepath.Join(hostRemoteTmpDir, host, inst.Name),
		)
		r.InstanceDeployDone(true)
	}
}

func (d *Deployer) parseTemplate(filename string) *template.Template {
	if t, ok := d.parsedTemplates[filename]; ok {
		return t
	}
	fileContents, err := os.ReadFile(filename)
	if err != nil {
		panic(rerrors.NewErrIo(filename, "parseTemplate", err))
	}
	t, err := rtemplate.New(filename).Option("missingkey=error").Parse(string(fileContents))
	if err != nil {
		panic(rtemplate.NewErrParse(filename, err))
	}
	d.parsedTemplates[filename] = t
	return t
}

func (d *Deployer) packInstance(inst *inventory.Instance, r *SingleReport) {
	fmt.Fprintf(os.Stderr, "Packing instance %s\n", inst.Name)
	r.InstancePackingStarted()
	defer r.InstancePackingDone()

	app := inst.App
	appFiles, err := fsys.GetAllFilesRecursive(filepath.Join("apps", app))
	if err != nil {
		panic(err)
	}
	instDir := filepath.Join(d.localTmpDir, inst.Host, inst.Name)
	err = os.Mkdir(instDir, 0755)
	if err != nil {
		panic(err)
	}
	for _, file := range appFiles {
		dstFile := strings.Replace(file, filepath.Join("apps", inst.App), instDir, 1)

		isTemplate := strings.HasSuffix(file, ".gotmpl")
		if isTemplate {
			dstFile = dstFile[0 : len(dstFile)-len(".gotmpl")]
		} else {
			isIgnoredTemplate := strings.HasSuffix(file, ".gotmpl_")
			if isIgnoredTemplate {
				dstFile = dstFile[0 : len(dstFile)-len("_")]
			}
		}

		dstFileDir := filepath.Dir(dstFile)
		os.MkdirAll(dstFileDir, 0755)
		dstFileHandle, err := os.Create(dstFile)
		if err != nil {
			panic(err)
		}
		defer dstFileHandle.Close()
		if !isTemplate {
			f, err := os.Open(file)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			_, err = io.Copy(dstFileHandle, f)
			if err != nil {
				panic(rerrors.NewErrIo(
					fmt.Sprintf("%s OR %s", file, dstFile),
					fmt.Sprintf("Copying %s to %s", file, dstFile),
					err,
				),
				)
			}
			continue
		}
		t := d.parseTemplate(file)
		err = t.Execute(dstFileHandle, d.resolvedInstanceVars[inst.Name])
		if err != nil {
			panic(rtemplate.NewErrExec(dstFile, fmt.Sprintf("packInstance %s", inst.Name), err))
		}
		continue
	}

	sh.MustDoSilentlyf("tar -C %s -cvf %s.tar .", instDir, instDir)

	if err := os.RemoveAll(instDir); err != nil {
		panic(err)
	}
}
