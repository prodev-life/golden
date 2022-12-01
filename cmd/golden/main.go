package main

import (
	"appcfg/pkg/deployer"
	"appcfg/pkg/inventory"
	"appcfg/pkg/manifest"
	"appcfg/pkg/rerrors"
	"appcfg/pkg/resolver"
	"appcfg/pkg/rtemplate"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
)


func main() {
	rootDirArg := pflag.StringP("root-dir", "r", ".", "directory with apps, manifests, instances, *_vars and others")
	manifNameArg := pflag.StringP("manifest", "m", "",
		"manifest name from manifests.yml or any of manifests/**.yml.\nCannot be specified with \"group\" argument.",
	)
	groupNameArg := pflag.StringP("group", "g", "",
		"group name to deploy.\nDeploys all instances that are part of this group.\nCannot be specified with \"manifest\" argument.",
	)
	appsArg := pflag.StringSliceP("apps", "a", []string{},
		"apps to deploy, comma separated.\nLimits apps to deploy within a specified group or manifest to those listed in this argument.",
	)
	locallyArg := pflag.BoolP("locally", "l", false,
		"ignore ssh* instructions for hosts and deploys all files locally\nto --local-prefix/_install_prefix_ which MUST be specifed.",
	)
	localPrefixTemplate := pflag.StringP("local-prefix", "p", "",
		"in conjuciton with --locally deploys all files to this prefix.\nCan be a template with all variables available.",
	)

	pflag.Parse()

	if *groupNameArg == "" && *manifNameArg == "" {
		fmt.Fprintln(os.Stderr, "Either --manifest or --group must be specified")
		pflag.Usage()
		os.Exit(1)
	}

	if *locallyArg && *localPrefixTemplate == "" {
		fmt.Fprintln(os.Stderr, "--locally requires --local-prefix to be specified")
		pflag.Usage()
		os.Exit(1)
	}

	var rep *deployer.Report
	var timeSpentOnResolving time.Duration

	defer func() {
		recovered := recover()
		ok := true
		rerrors.Recover(recovered, &ok)
		if rep != nil {
			fmt.Fprintf(os.Stderr, "Spent on resolving variables: %s\n", timeSpentOnResolving.String())
			fmt.Fprint(os.Stderr, rep.String())
		}
		if ok {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}()

	err := os.Chdir(*rootDirArg)
	if err != nil {
		panic(err)
	}

	resolvingStarted := time.Now()
	inv := inventory.ReadInventory("")
	r := resolver.New("", inv)

	if *locallyArg {
		inv.SetHostsToLocalhost()
		insts := inv.GetAllInstances()
		overrides := map[string]string{}
		tmpl, err := rtemplate.New("local-prefix").Parse(*localPrefixTemplate)
		if err != nil {
			panic(rtemplate.NewErrParse("--local-prefix", err))
		}
		for k, v := range insts {
			vars := resolver.CreateBuiltInVars(v)
			preparedVars := vars.SubstituteTemplatedVars()
			prefix, err := rtemplate.ExecToString(tmpl, preparedVars)
			if err != nil {
				panic(rtemplate.NewErrExec("--local-prefix", "Executing --local-prefix template", err))
			}
			overrides[k] = filepath.Join(prefix, v.InstallPrefix)
		}
		inv.OverrideInstallPrefix(overrides)
	}

	manifests := manifest.ReadManifestsCollection("manifests")

	var manif *manifest.Manifest
	var ok bool
	if *manifNameArg != "" {
		manif, ok = manifests[*manifNameArg]
		if !ok {
			panic(rerrors.NewErrStringf("--manifest %s does not exist", *manifNameArg))
		}
	} else {
		manif = &manifest.Manifest{*groupNameArg}
	}

	appsWhiteList := map[string]struct{}{}
	for _, app := range *appsArg {
		appsWhiteList[app] = struct{}{}
	}

	for _, inst := range inv.GetInstancesForManifest(*manif) {
		if len(appsWhiteList) > 0 {
			if _, ok := appsWhiteList[inst.App]; !ok {
				continue
			}
		}

		r.ResolveInstance(inst)
	}

	timeSpentOnResolving = time.Since(resolvingStarted)

	d := deployer.New(r.GetAllResolvedVars(), inv)
	rep = d.Deploy(*manif, *appsArg)
}
