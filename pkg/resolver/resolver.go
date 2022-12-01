package resolver

import (
	"appcfg/pkg/inventory"
	"appcfg/pkg/varmap"
	"path/filepath"
)

type VarSource int
const (
	VarSourceCommon VarSource = iota
	VarSourceGroup
	VarSourceHost
	VarSourceApp
	VarSourceInstance
)


func New(rootDir string, inv *inventory.Inventory) *Resolver {
	r := &Resolver{
		rootDir: rootDir,
		inv: inv,
	}
	return r
}

type Resolver struct {
	rootDir string
	inv *inventory.Inventory
	commonVars varmap.VarMap
	groupVars map[string]varmap.VarMap
	hostVars map[string]varmap.VarMap
	appVars map[string]varmap.VarMap
	instanceVars map[string]varmap.VarMap

	finalInstanceVars map[string]map[string]interface{}
}

func (r *Resolver) GetAllResolvedVars() map[string]map[string]interface{} {
	return r.finalInstanceVars
}

func (r *Resolver) GetResolvedVars(instanceName string) map[string]interface{} {
	return r.finalInstanceVars[instanceName]
}

func CreateBuiltInVars(inst *inventory.Instance) varmap.VarMap {
	m := varmap.New()
	m["_host_"] = &varmap.Var{Value:inst.Host}
	m["_app_"] = &varmap.Var{Value:inst.App}
	m["_instance_"] = &varmap.Var{Value: inst.Name}
	m["_install_prefix_"] = &varmap.Var{Value: inst.InstallPrefix}
	m.SetSource("_builtin_")
	return m
}

func (r *Resolver) ResolveInstance(inst *inventory.Instance) (finalVars map[string]interface{}) {
	if r.finalInstanceVars == nil {
		r.finalInstanceVars = make(map[string]map[string]interface{})
	}
	ok := false
	if finalVars, ok = r.finalInstanceVars[inst.Name]; ok {
		return finalVars
	}

	defer func() {
		r.finalInstanceVars[inst.Name] = finalVars
	}()

	vars := varmap.New()
	vars = varmap.Merge(vars, r.getCommonVars(), varmap.ConflictResolutionOverride)
	vars = varmap.Merge(vars, r.getAppVars(inst.App), varmap.ConflictResolutionOverride)

	groups := r.inv.GetGroups(inst.Name)
	grMap := varmap.New()
	for _, gr := range groups {
		grMap = varmap.Merge(grMap, r.getGroupVars(gr), varmap.ConflictResolutionError)
	}

	vars = varmap.Merge(vars, grMap, varmap.ConflictResolutionOverride)

	host := inst.Host
	vars = varmap.Merge(vars, r.getHostVars(host), varmap.ConflictResolutionOverride)
	vars = varmap.Merge(vars, r.getInstanceVars(inst.Name), varmap.ConflictResolutionOverride)
	vars = varmap.Merge(vars, CreateBuiltInVars(inst), varmap.ConflictResolutionError)

	finalVars = vars.SubstituteTemplatedVars()

	return finalVars
}

func (r *Resolver) getCommonVars() varmap.VarMap {
	if r.commonVars == nil {
		r.commonVars = varmap.Read("common_vars")
	}
	return r.commonVars
}

func (r *Resolver) getGroupVars(group string) varmap.VarMap {
	if r.groupVars == nil {
		r.groupVars = make(map[string]varmap.VarMap)
	}
	if _, ok := r.groupVars[group]; !ok {
		m := varmap.Read(filepath.Join("group_vars", group))
		r.groupVars[group] = m
		return m
	}
	return r.groupVars[group]
}

func (r *Resolver) getHostVars(host string) varmap.VarMap {
	if r.hostVars == nil {
		r.hostVars = make(map[string]varmap.VarMap)
	}
	if _, ok := r.hostVars[host]; !ok {
		m := varmap.Read(filepath.Join("host_vars", host))
		r.hostVars[host] = m
		return m

	}
	return r.hostVars[host]
}

func (r *Resolver) getAppVars(app string) varmap.VarMap {
	if r.appVars == nil {
		r.appVars = make(map[string]varmap.VarMap)
	}
	if _, ok := r.appVars[app]; !ok {
		m := varmap.Read(filepath.Join("app_vars", app))
		r.appVars[app] = m
		return m

	}
	return r.appVars[app]
}

func (r *Resolver) getInstanceVars(inst string) varmap.VarMap {
	if r.instanceVars == nil {
		r.instanceVars = make(map[string]varmap.VarMap)
	}
	if _, ok := r.instanceVars[inst]; !ok {
		m := varmap.Read(filepath.Join("instance_vars", inst))
		r.instanceVars[inst] = m
		return m

	}
	return r.instanceVars[inst]
}