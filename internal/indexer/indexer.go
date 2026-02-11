package indexer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	appslisters "k8s.io/client-go/listers/apps/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Indexer struct {
	factory informers.SharedInformerFactory

	rolesInformer               cache.SharedIndexInformer
	clusterRolesInformer        cache.SharedIndexInformer
	roleBindingsInformer        cache.SharedIndexInformer
	clusterRoleBindingsInformer cache.SharedIndexInformer
	podsInformer                cache.SharedIndexInformer
	deploymentsInformer         cache.SharedIndexInformer
	replicaSetsInformer         cache.SharedIndexInformer
	statefulSetsInformer        cache.SharedIndexInformer
	daemonSetsInformer          cache.SharedIndexInformer
	jobsInformer                cache.SharedIndexInformer
	cronJobsInformer            cache.SharedIndexInformer

	rolesLister               rbaclisters.RoleLister
	clusterRolesLister        rbaclisters.ClusterRoleLister
	roleBindingsLister        rbaclisters.RoleBindingLister
	clusterRoleBindingsLister rbaclisters.ClusterRoleBindingLister
	podsLister                corelisters.PodLister
	deploymentsLister         appslisters.DeploymentLister
	replicaSetsLister         appslisters.ReplicaSetLister
	statefulSetsLister        appslisters.StatefulSetLister
	daemonSetsLister          appslisters.DaemonSetLister
	jobsLister                batchlisters.JobLister
	cronJobsLister            batchlisters.CronJobLister
	snapshot                  atomic.Pointer[Snapshot]
	synced                    atomic.Bool
	rebuildMu                 sync.Mutex
}

func New(client kubernetes.Interface, resyncPeriod time.Duration) *Indexer {
	factory := informers.NewSharedInformerFactory(client, resyncPeriod)
	roles := factory.Rbac().V1().Roles()
	clusterRoles := factory.Rbac().V1().ClusterRoles()
	roleBindings := factory.Rbac().V1().RoleBindings()
	clusterRoleBindings := factory.Rbac().V1().ClusterRoleBindings()
	pods := factory.Core().V1().Pods()
	deployments := factory.Apps().V1().Deployments()
	replicaSets := factory.Apps().V1().ReplicaSets()
	statefulSets := factory.Apps().V1().StatefulSets()
	daemonSets := factory.Apps().V1().DaemonSets()
	jobs := factory.Batch().V1().Jobs()
	cronJobs := factory.Batch().V1().CronJobs()

	i := &Indexer{
		factory:                     factory,
		rolesInformer:               roles.Informer(),
		clusterRolesInformer:        clusterRoles.Informer(),
		roleBindingsInformer:        roleBindings.Informer(),
		clusterRoleBindingsInformer: clusterRoleBindings.Informer(),
		podsInformer:                pods.Informer(),
		deploymentsInformer:         deployments.Informer(),
		replicaSetsInformer:         replicaSets.Informer(),
		statefulSetsInformer:        statefulSets.Informer(),
		daemonSetsInformer:          daemonSets.Informer(),
		jobsInformer:                jobs.Informer(),
		cronJobsInformer:            cronJobs.Informer(),
		rolesLister:                 roles.Lister(),
		clusterRolesLister:          clusterRoles.Lister(),
		roleBindingsLister:          roleBindings.Lister(),
		clusterRoleBindingsLister:   clusterRoleBindings.Lister(),
		podsLister:                  pods.Lister(),
		deploymentsLister:           deployments.Lister(),
		replicaSetsLister:           replicaSets.Lister(),
		statefulSetsLister:          statefulSets.Lister(),
		daemonSetsLister:            daemonSets.Lister(),
		jobsLister:                  jobs.Lister(),
		cronJobsLister:              cronJobs.Lister(),
	}

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc:    func(any) { i.rebuild() },
		UpdateFunc: func(any, any) { i.rebuild() },
		DeleteFunc: func(any) { i.rebuild() },
	}
	i.rolesInformer.AddEventHandler(handler)
	i.clusterRolesInformer.AddEventHandler(handler)
	i.roleBindingsInformer.AddEventHandler(handler)
	i.clusterRoleBindingsInformer.AddEventHandler(handler)
	i.podsInformer.AddEventHandler(handler)
	i.deploymentsInformer.AddEventHandler(handler)
	i.replicaSetsInformer.AddEventHandler(handler)
	i.statefulSetsInformer.AddEventHandler(handler)
	i.daemonSetsInformer.AddEventHandler(handler)
	i.jobsInformer.AddEventHandler(handler)
	i.cronJobsInformer.AddEventHandler(handler)

	i.snapshot.Store(newEmptySnapshot())
	return i
}

func (i *Indexer) Start(ctx context.Context) error {
	i.factory.Start(ctx.Done())

	if !cache.WaitForCacheSync(
		ctx.Done(),
		i.rolesInformer.HasSynced,
		i.clusterRolesInformer.HasSynced,
		i.roleBindingsInformer.HasSynced,
		i.clusterRoleBindingsInformer.HasSynced,
		i.podsInformer.HasSynced,
		i.deploymentsInformer.HasSynced,
		i.replicaSetsInformer.HasSynced,
		i.statefulSetsInformer.HasSynced,
		i.daemonSetsInformer.HasSynced,
		i.jobsInformer.HasSynced,
		i.cronJobsInformer.HasSynced,
	) {
		return fmt.Errorf("failed to sync informer caches")
	}

	i.rebuild()
	i.synced.Store(true)
	<-ctx.Done()
	return nil
}

func (i *Indexer) IsReady() bool {
	return i.synced.Load()
}

func (i *Indexer) Snapshot() *Snapshot {
	s := i.snapshot.Load()
	if s == nil {
		return newEmptySnapshot()
	}
	return s
}

func (i *Indexer) rebuild() {
	i.rebuildMu.Lock()
	defer i.rebuildMu.Unlock()

	next := newEmptySnapshot()
	next.BuiltAt = time.Now().UTC()

	roles := listWithWarning(i.rolesLister.List, "roles", &next.Warnings)
	clusterRoles := listWithWarning(i.clusterRolesLister.List, "clusterroles", &next.Warnings)
	roleBindings := listWithWarning(i.roleBindingsLister.List, "rolebindings", &next.Warnings)
	clusterRoleBindings := listWithWarning(i.clusterRoleBindingsLister.List, "clusterrolebindings", &next.Warnings)
	pods := listWithWarning(i.podsLister.List, "pods", &next.Warnings)
	deployments := listWithWarning(i.deploymentsLister.List, "deployments", &next.Warnings)
	replicaSets := listWithWarning(i.replicaSetsLister.List, "replicasets", &next.Warnings)
	statefulSets := listWithWarning(i.statefulSetsLister.List, "statefulsets", &next.Warnings)
	daemonSets := listWithWarning(i.daemonSetsLister.List, "daemonsets", &next.Warnings)
	jobs := listWithWarning(i.jobsLister.List, "jobs", &next.Warnings)
	cronJobs := listWithWarning(i.cronJobsLister.List, "cronjobs", &next.Warnings)

	indexRoles(next, roles)
	indexClusterRoles(next, clusterRoles)
	indexAggregatedClusterRoles(next, clusterRoles)
	indexRoleBindings(next, roleBindings)
	indexClusterRoleBindings(next, clusterRoleBindings)

	indexPods(next, pods)
	for _, deployment := range deployments {
		indexWorkload(next, deployment.UID, "apps/v1", "Deployment", deployment.Namespace, deployment.Name, deployment.OwnerReferences)
	}
	for _, replicaSet := range replicaSets {
		indexWorkload(next, replicaSet.UID, "apps/v1", "ReplicaSet", replicaSet.Namespace, replicaSet.Name, replicaSet.OwnerReferences)
	}
	for _, statefulSet := range statefulSets {
		indexWorkload(next, statefulSet.UID, "apps/v1", "StatefulSet", statefulSet.Namespace, statefulSet.Name, statefulSet.OwnerReferences)
	}
	for _, daemonSet := range daemonSets {
		indexWorkload(next, daemonSet.UID, "apps/v1", "DaemonSet", daemonSet.Namespace, daemonSet.Name, daemonSet.OwnerReferences)
	}
	for _, job := range jobs {
		indexWorkload(next, job.UID, "batch/v1", "Job", job.Namespace, job.Name, job.OwnerReferences)
	}
	for _, cronJob := range cronJobs {
		indexWorkload(next, cronJob.UID, "batch/v1", "CronJob", cronJob.Namespace, cronJob.Name, cronJob.OwnerReferences)
	}

	sortSnapshot(next)
	i.snapshot.Store(next)
}
