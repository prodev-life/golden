package main

import (
	"fmt"
	"golden/pkg/deployer"
	"golden/pkg/git"
	"golden/pkg/inventory"
	"golden/pkg/manifest"
	"golden/pkg/rerrors"
	"golden/pkg/resolver"
	"golden/pkg/rtemplate"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
)


func main() {
	versionArg := pflag.BoolP("version", "v", false, "displays version of golden")
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
	installPrefixTemplateArg := pflag.StringP("prefix", "p", "",
		`Prepends --prefix to install_prefix for instances.
		E. g. in conjuciton with --locally deploys all files locally to this --prefix.
		Can be a template with builtin variables available.`,
	)

	pflag.Parse()

	if *versionArg {
		fmt.Printf("golden version: %s\n", git.Version)
		os.Exit(0)
	}

	if *groupNameArg == "" && *manifNameArg == "" {
		fmt.Fprintln(os.Stderr, "Either --manifest or --group must be specified")
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
	}
	if *installPrefixTemplateArg != "" {
		insts := inv.GetAllInstances()
		overrides := map[string]string{}
		tmpl, err := rtemplate.New("--prefix").Parse(*installPrefixTemplateArg)
		if err != nil {
			panic(rtemplate.NewErrParse("--prefix", err))
		}
		for k, v := range insts {
			vars := resolver.CreateBuiltInVars(v)
			preparedVars, substError := vars.SubstituteTemplatedVars()
			if substError != nil {
				panic(err)
			}
			prefix, err := rtemplate.ExecToString(tmpl, preparedVars)
			if err != nil {
				panic(rtemplate.NewErrExec("--prefix", "Executing --prefix template", err))
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
	resolvedVars, substitutionErrors := r.GetAllResolvedVarsAndErrors()
	d := deployer.New(resolvedVars, substitutionErrors, inv)
	rep = d.Deploy(*manif, *appsArg)
}
