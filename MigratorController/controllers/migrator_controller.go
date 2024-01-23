/*
Copyright 2023 Jiaxuan Chen.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/migrator/controllers/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	migrationv1beta1 "github.com/migrator/api/v1beta1"
)

const (
	filerPodLabel  = "ownerMigrator"
	initialPodFlag = true
	targetPodFlag  = true
)

// MigratorReconciler reconciles a Migrator object
type MigratorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=migration.bupt.cjx,resources=migrators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=migration.bupt.cjx,resources=migrators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=migration.bupt.cjx,resources=migrators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Migrator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *MigratorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	ctx = context.Background()

	// Fetching Migrator
	migrator := &migrationv1beta1.Migrator{}
	if err := r.Get(ctx, req.NamespacedName, migrator); err != nil {
		if err := client.IgnoreNotFound(err); err != nil {
			logger.Error(err, "fetch Migrator failed")
			return ctrl.Result{}, err
		}
	}
	// Getting sub pod of migrator
	subPodList, err := r.getSubPods(ctx, req.Namespace, req.Name)

	if err != nil {
		logger.Error(err, "fetch subPods failed")
		return ctrl.Result{}, nil
	}
	sourcePod, targetPod := r.getSourceTargetPods(subPodList)

	// Update State --------------------------------------
	// Update migrator state to "CreatingSourcePod"
	creationTime := migrator.ObjectMeta.CreationTimestamp
	timePassed := time.Since(creationTime.Time)
	if len(subPodList.Items) == 0 &&
		migrator.Status.MigrationState == "" &&
		migrator.DeletionTimestamp == nil &&
		timePassed.Seconds() < 5 { //这个DeletionTimestamp条件没用！删除的时候还是会进入到这个if中
		// Update migrator state to "CreatingSourcePod".
		if err := r.updateMigratorStatusState(ctx, migrator, migrationv1beta1.StateCreatingSourcePod); err != nil {
			logger.Error(err, "Update migrator state failed")
			return ctrl.Result{}, err
		}
		logger.Info("Update state successful: CreatingSourcePod")
	}

	// Update migrator state to  "Running"             //此处缺垃圾回收之后的条件，目前只是源pod running的条件
	if targetPod == nil &&
		sourcePod != nil &&
		migrator.Spec.MigrationTrigger != true &&
		sourcePod.Status.Phase == corev1.PodRunning &&
		(migrator.Status.MigrationState == migrationv1beta1.StateCreatingSourcePod ||
			migrator.Status.MigrationState == migrationv1beta1.StateMigrated) {
		if err := r.updateMigratorStatusState(ctx, migrator, migrationv1beta1.StateRunning); err != nil {
			logger.Error(err, "Update migrator state failed")
			return ctrl.Result{}, err
		}
		logger.Info("Update state successful: Running")
	}

	// Update migrator state to "Migrating"
	if migrator.Spec.MigrationTrigger == true &&
		migrator.Status.MigrationState == migrationv1beta1.StateRunning {
		if migrator.Spec.TargetNode == "" {
			migrator.Spec.MigrationTrigger = false
			logger.Info("Warning! Please config target Pod name!")
			if err := r.Update(ctx, migrator); err != nil {
				logger.Error(err, "Update migrator migrator.Spec.MigrationTrigger failed")
			}
		} else {
			if err := r.updateMigratorStatusState(ctx, migrator, migrationv1beta1.StateMigrating); err != nil {
				logger.Error(err, "Update migrator state failed")
				return ctrl.Result{}, err
			}
			logger.Info("Update state successful: Migrating")
		}
	}

	// Update migrator state to "StateMigrated"
	if migrator.Spec.MigrationTrigger == true &&
		migrator.Status.MigrationState == migrationv1beta1.StateMigrating &&
		targetPod != nil &&
		targetPod.Status.Phase == "Running" {
		if err := r.updateMigratorStatusState(ctx, migrator, migrationv1beta1.StateMigrated); err != nil {
			logger.Error(err, "Update migrator state failed")
			return ctrl.Result{}, err
		}
		logger.Info("Update state successful: Migrated")
	}

	// CRUD --------------------------------------
	// Create newSourcePod. （&& len(subPodList.Items) --→ Prevent multiple creation）
	if migrator.Status.MigrationState == migrationv1beta1.StateCreatingSourcePod && len(subPodList.Items) == 0 {
		logger.Info("Creating newSourcePod")
		newSourcePod := utils.GetPodFromTemplate(&migrator.Spec.Template, req.Namespace)
		// Get and set podName which will be created.
		newSourcePod.Name, err = utils.GetPodName(migrator.Name, "", initialPodFlag)
		if err != nil {
			logger.Error(err, "Get PodName failed, check name format")
		}
		// Set Pod owner.
		if err := ctrl.SetControllerReference(migrator, newSourcePod, r.Scheme); err != nil {
			logger.Error(err, "Set ControllerReference failed")
			return ctrl.Result{}, err
		}
		// Create pod!
		err := r.Create(ctx, newSourcePod)
		if err != nil {
			logger.Error(err, "create newSourcePod failed")
			return ctrl.Result{}, err
		}
		// Update SourcePod Name.
		err = r.updateMigratorStatusSourceTarget(ctx, migrator, newSourcePod.Name, "")
		if err != nil {
			logger.Error(err, "Update SourcePodName failed")
			return ctrl.Result{}, err
		}
	}

	// Create targetPod and begin to migration.
	if migrator.Status.MigrationState == migrationv1beta1.StateMigrating && len(subPodList.Items) == 1 {
		newTargetPod := utils.GetPodFromTemplate(&migrator.Spec.Template, req.Namespace)
		// Get targetPod Name.
		newTargetPod.Name, err = utils.GetPodName(migrator.Name, sourcePod.Name, !initialPodFlag)
		if err != nil {
			logger.Error(err, "Get PodName failed, check name format")
		}
		newTargetPod.Labels["CloneSourcePod"] = sourcePod.Name
		newTargetPod.Spec.NodeName = migrator.Spec.TargetNode
		newTargetPod.Spec.InitContainers = nil
		// Set Pod owner.
		if err := ctrl.SetControllerReference(migrator, newTargetPod, r.Scheme); err != nil {
			logger.Error(err, "Set ControllerReference failed")
			return ctrl.Result{}, err
		}
		// Update TargetPod Name.
		err = r.updateMigratorStatusSourceTarget(ctx, migrator, sourcePod.Name, newTargetPod.Name)
		if err != nil {
			logger.Error(err, "Update TargetPodName failed")
			return ctrl.Result{}, err
		}
		// Update sourcePod labels.
		if sourcePod.Labels == nil {
			sourcePod.Labels = make(map[string]string)
		}
		sourcePod.Labels["MigratingTo"] = migrator.Spec.TargetNode
		if err := r.Update(ctx, sourcePod); err != nil {
			logger.Error(err, "update sourcePod labels failed")
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, newTargetPod); errors.IsNotFound(err) {
			logger.Info("Creating TargetPod and preparing for migration")
			// Create targetPod!
			if err := r.Create(ctx, newTargetPod); err != nil {
				logger.Error(err, "create newTargetPod failed")
				return ctrl.Result{}, err
			}
		}
	}

	// Garbage collection and Clean env.
	if migrator.Status.MigrationState == migrationv1beta1.StateMigrated && len(subPodList.Items) == 2 && migrator.Spec.MigrationTrigger == true {
		// Switch the migration trigger state
		migrator.Spec.MigrationTrigger = false
		// Delete target Node
		migrator.Spec.TargetNode = ""
		// Update migrator
		if err := r.Update(ctx, migrator); err != nil {
			logger.Error(err, "Update migrator failed")
		}

		// delete targetPod label
		delete(targetPod.Labels, "CloneSourcePod")
		// Update targetPod!
		if err := r.Update(ctx, targetPod); err != nil {
			logger.Error(err, "Update targetPod failed")
		}

		// Delete sourcePod
		logger.Info("Deleting sourcePod")
		if err := r.Delete(ctx, sourcePod); err != nil {
			logger.Error(err, "Delete sourcePod failed")
			return ctrl.Result{}, err
		}
		// Update targetPod to sourcePod
		if err := r.updateMigratorStatusSourceTarget(ctx, migrator, targetPod.Name, ""); err != nil {
			logger.Error(err, "Update SourceTargetName failed")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MigratorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&migrationv1beta1.Migrator{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
func (r *MigratorReconciler) getSubPods(ctx context.Context, migratorNamespace string, migratorName string) (*corev1.PodList, error) {
	pods := &corev1.PodList{}
	subPodList := &corev1.PodList{}
	if err := r.List(ctx, pods, client.InNamespace(migratorNamespace)); err != nil {
		return nil, err
	}
	for _, pod := range pods.Items {
		for _, owner := range pod.OwnerReferences {
			if owner.Name == migratorName {
				subPodList.Items = append(subPodList.Items, pod)
			}
		}
	}
	return subPodList, nil
}

func (r *MigratorReconciler) getSourceTargetPods(subPods *corev1.PodList) (sourcePod *corev1.Pod, targetPod *corev1.Pod) {
	if len(subPods.Items) == 1 {
		return &subPods.Items[0], nil
	}
	if len(subPods.Items) == 2 {
		if subPods.Items[0].CreationTimestamp.Before(&subPods.Items[1].CreationTimestamp) {
			return &subPods.Items[0], &subPods.Items[1]
		} else {
			return &subPods.Items[1], &subPods.Items[0]
		}
	}
	return nil, nil
}

func (r *MigratorReconciler) updateMigratorStatusState(ctx context.Context, migrator *migrationv1beta1.Migrator, state string) error {
	migrator.Status.MigrationState = state
	if err := r.Status().Update(ctx, migrator); err != nil {
		return nil
	} else {
		return err
	}
}

func (r *MigratorReconciler) updateMigratorStatusSourceTarget(ctx context.Context, migrator *migrationv1beta1.Migrator, sourcePodName string, targetPodName string) error {
	migrator.Status.SourcePod = sourcePodName
	migrator.Status.TargetPod = targetPodName
	if err := r.Status().Update(ctx, migrator); err != nil {
		return nil
	} else {
		return err
	}
}
