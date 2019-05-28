package views

import (
	"github.com/derailed/k9s/internal/k8s"
	"github.com/derailed/k9s/internal/resource"
	"github.com/gdamore/tcell"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	viewFn     func(ns string, app *appView, list resource.List) resourceViewer
	listFn     func(c resource.Connection, ns string) resource.List
	colorerFn  func(ns string, evt *resource.RowEvent) tcell.Color
	enterFn    func(app *appView, ns, resource, selection string)
	decorateFn func(resource.TableData) resource.TableData

	resCmd struct {
		title      string
		api        string
		viewFn     viewFn
		listFn     listFn
		enterFn    enterFn
		colorerFn  colorerFn
		decorateFn decorateFn
	}
)

func helpCmds(c k8s.Connection) map[string]resCmd {
	cmdMap := resourceViews(c)
	cmds := make(map[string]resCmd, len(cmdMap))
	for k, v := range cmdMap {
		cmds[k] = v
	}
	for k, v := range allCRDs(c) {
		cmds[k] = resCmd{title: v.Kind, api: v.Group}
	}

	return cmds
}

func allCRDs(c k8s.Connection) map[string]k8s.APIGroup {
	m := map[string]k8s.APIGroup{}

	crds, _ := resource.NewCustomResourceDefinitionList(c, resource.AllNamespaces).
		Resource().
		List(resource.AllNamespaces)

	for _, crd := range crds {
		ff := crd.ExtFields()

		grp := k8s.APIGroup{
			Group:   ff["group"].(string),
			Kind:    ff["kind"].(string),
			Version: ff["version"].(string),
		}

		if p, ok := ff["plural"].(string); ok {
			grp.Plural = p
			m[p] = grp
		}

		if s, ok := ff["singular"].(string); ok {
			grp.Singular = s
			m[s] = grp
		}

		if aa, ok := ff["aliases"].([]interface{}); ok {
			for _, a := range aa {
				m[a.(string)] = grp
			}
		}
	}

	return m
}

func showRBAC(app *appView, ns, resource, selection string) {
	kind := clusterRole
	if resource == "role" {
		kind = role
	}
	app.inject(newRBACView(app, ns, selection, kind))
}

func showClusterRole(app *appView, ns, resource, selection string) {
	crb, err := app.conn().DialOrDie().Rbac().ClusterRoleBindings().Get(selection, metav1.GetOptions{})
	if err != nil {
		app.flash().errf("Unable to retrieve clusterrolebindings for %s", selection)
		return
	}
	app.inject(newRBACView(app, ns, crb.RoleRef.Name, clusterRole))
}

func showRole(app *appView, _, resource, selection string) {
	ns, n := namespaced(selection)
	rb, err := app.conn().DialOrDie().Rbac().RoleBindings(ns).Get(n, metav1.GetOptions{})
	if err != nil {
		app.flash().errf("Unable to retrieve rolebindings for %s", selection)
		return
	}
	app.inject(newRBACView(app, ns, fqn(ns, rb.RoleRef.Name), role))
}

func showSAPolicy(app *appView, _, _, selection string) {
	_, n := namespaced(selection)
	app.inject(newPolicyView(app, mapFuSubject("ServiceAccount"), n))
}

func resourceViews(c k8s.Connection) map[string]resCmd {
	cmds := map[string]resCmd{
		"cm": {
			title:  "ConfigMaps",
			api:    "",
			viewFn: newResourceView,
			listFn: resource.NewConfigMapList,
		},
		"cr": {
			title:   "ClusterRoles",
			api:     "rbac.authorization.k8s.io",
			viewFn:  newResourceView,
			listFn:  resource.NewClusterRoleList,
			enterFn: showRBAC,
		},
		"crb": {
			title:   "ClusterRoleBindings",
			api:     "rbac.authorization.k8s.io",
			viewFn:  newResourceView,
			listFn:  resource.NewClusterRoleBindingList,
			enterFn: showClusterRole,
		},
		"crd": {
			title:  "CustomResourceDefinitions",
			api:    "apiextensions.k8s.io",
			viewFn: newResourceView,
			listFn: resource.NewCustomResourceDefinitionList,
		},
		"cj": {
			title:  "CronJobs",
			api:    "batch",
			viewFn: newCronJobView,
			listFn: resource.NewCronJobList,
		},
		"ctx": {
			title:     "Contexts",
			api:       "",
			viewFn:    newContextView,
			listFn:    resource.NewContextList,
			colorerFn: ctxColorer,
		},
		"ds": {
			title:     "DaemonSets",
			api:       "",
			viewFn:    newDaemonSetView,
			listFn:    resource.NewDaemonSetList,
			colorerFn: dpColorer,
		},
		"dp": {
			title:     "Deployments",
			api:       "apps",
			viewFn:    newDeployView,
			listFn:    resource.NewDeploymentList,
			colorerFn: dpColorer,
		},
		"ep": {
			title:  "EndPoints",
			api:    "",
			viewFn: newResourceView,
			listFn: resource.NewEndpointsList,
		},
		"ev": {
			title:     "Events",
			api:       "",
			viewFn:    newResourceView,
			listFn:    resource.NewEventList,
			colorerFn: evColorer,
		},
		"ing": {
			title:  "Ingress",
			api:    "extensions",
			viewFn: newResourceView,
			listFn: resource.NewIngressList,
		},
		"jo": {
			title:  "Jobs",
			api:    "batch",
			viewFn: newJobView,
			listFn: resource.NewJobList,
		},
		"no": {
			title:     "Nodes",
			api:       "",
			viewFn:    newNodeView,
			listFn:    resource.NewNodeList,
			colorerFn: nsColorer,
		},
		"ns": {
			title:     "Namespaces",
			api:       "",
			viewFn:    newNamespaceView,
			listFn:    resource.NewNamespaceList,
			colorerFn: nsColorer,
		},
		"pdb": {
			title:     "PodDisruptionBudgets",
			api:       "v1.beta1",
			viewFn:    newResourceView,
			listFn:    resource.NewPDBList,
			colorerFn: pdbColorer,
		},
		"po": {
			title:     "Pods",
			api:       "",
			viewFn:    newPodView,
			listFn:    resource.NewPodList,
			colorerFn: podColorer,
		},
		"pv": {
			title:     "PersistentVolumes",
			api:       "",
			viewFn:    newResourceView,
			listFn:    resource.NewPersistentVolumeList,
			colorerFn: pvColorer,
		},
		"pvc": {
			title:     "PersistentVolumeClaims",
			api:       "",
			viewFn:    newResourceView,
			listFn:    resource.NewPersistentVolumeClaimList,
			colorerFn: pvcColorer,
		},
		"rb": {
			title:   "RoleBindings",
			api:     "rbac.authorization.k8s.io",
			viewFn:  newResourceView,
			listFn:  resource.NewRoleBindingList,
			enterFn: showRole,
		},
		"rc": {
			title:     "ReplicationControllers",
			api:       "",
			viewFn:    newResourceView,
			listFn:    resource.NewReplicationControllerList,
			colorerFn: rsColorer,
		},
		"ro": {
			title:   "Roles",
			api:     "rbac.authorization.k8s.io",
			viewFn:  newResourceView,
			listFn:  resource.NewRoleList,
			enterFn: showRBAC,
		},
		"rs": {
			title:     "ReplicaSets",
			api:       "apps",
			viewFn:    newReplicaSetView,
			listFn:    resource.NewReplicaSetList,
			colorerFn: rsColorer,
		},
		"sa": {
			title:   "ServiceAccounts",
			api:     "",
			viewFn:  newResourceView,
			listFn:  resource.NewServiceAccountList,
			enterFn: showSAPolicy,
		},
		"sec": {
			title:  "Secrets",
			api:    "",
			viewFn: newSecretView,
			listFn: resource.NewSecretList,
		},
		"sts": {
			title:     "StatefulSets",
			api:       "apps",
			viewFn:    newStatefulSetView,
			listFn:    resource.NewStatefulSetList,
			colorerFn: stsColorer,
		},
		"svc": {
			title:  "Services",
			api:    "",
			viewFn: newSvcView,
			listFn: resource.NewServiceList,
		},
		"usr": {
			title:  "Users",
			api:    "",
			viewFn: newSubjectView,
		},
		"grp": {
			title:  "Groups",
			api:    "",
			viewFn: newSubjectView,
		},
	}

	rev, ok, err := c.SupportsRes("autoscaling", []string{"v1", "v2beta1", "v2beta2"})
	if err != nil {
		log.Error().Err(err).Msg("Checking HPA")
		return cmds
	}
	if !ok {
		log.Error().Msg("HPA are not supported on this cluster")
		return cmds
	}

	switch rev {
	case "v1":
		cmds["hpa"] = resCmd{
			title:  "HorizontalPodAutoscalers",
			api:    "autoscaling",
			viewFn: newResourceView,
			listFn: resource.NewHorizontalPodAutoscalerV1List,
		}
	case "v2beta1":
		cmds["hpa"] = resCmd{
			title:  "HorizontalPodAutoscalers",
			api:    "autoscaling",
			viewFn: newResourceView,
			listFn: resource.NewHorizontalPodAutoscalerV2Beta1List,
		}
	case "v2beta2":
		cmds["hpa"] = resCmd{
			title:  "HorizontalPodAutoscalers",
			api:    "autoscaling",
			viewFn: newResourceView,
			listFn: resource.NewHorizontalPodAutoscalerList,
		}
	default:
		log.Panic().Msgf("K9s unsupported HPA version. Exiting!")
	}

	return cmds
}
