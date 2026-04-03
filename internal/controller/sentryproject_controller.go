package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	sentryv1alpha1 "github.com/agjmills/sentry-operator/api/v1alpha1"
	"github.com/agjmills/sentry-operator/internal/sentry"
)

const (
	finalizerName = "sentry-operator.io/finalizer"
)

// Config holds operator-level defaults, set via CLI flags.
type Config struct {
	DefaultOrganization   string
	DefaultTeam           string
	DefaultPlatform       string
	DefaultRetainOnDelete bool
	SentryURL             string
	RequeueInterval       time.Duration
}

// SentryProjectReconciler reconciles SentryProject objects.
type SentryProjectReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	SentryClient *sentry.Client
	Config       Config
}

// +kubebuilder:rbac:groups=sentry-operator.io,resources=sentryprojects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sentry-operator.io,resources=sentryprojects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sentry-operator.io,resources=sentryprojects/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *SentryProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var sp sentryv1alpha1.SentryProject
	if err := r.Get(ctx, req.NamespacedName, &sp); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !sp.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &sp)
	}

	if !controllerutil.ContainsFinalizer(&sp, finalizerName) {
		controllerutil.AddFinalizer(&sp, finalizerName)
		if err := r.Update(ctx, &sp); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	org := coalesce(sp.Spec.Organization, r.Config.DefaultOrganization)
	team := coalesce(sp.Spec.Team, r.Config.DefaultTeam)
	platform := coalesce(sp.Spec.Platform, r.Config.DefaultPlatform)
	projectSlug := coalesce(sp.Spec.ProjectSlug, sp.Name)
	secretName := coalesce(sp.Spec.SecretName, sp.Name+"-sentry")

	if org == "" || team == "" {
		return r.setFailed(ctx, &sp, "organization and team must be set (via spec or operator defaults)")
	}

	logger.Info("reconciling SentryProject", "org", org, "team", team, "slug", projectSlug)

	project, err := r.SentryClient.GetProject(ctx, org, projectSlug)
	if err != nil {
		return r.setFailed(ctx, &sp, fmt.Sprintf("get project: %v", err))
	}

	if project == nil {
		logger.Info("project not found, creating", "slug", projectSlug)
		project, err = r.SentryClient.CreateProject(ctx, org, team, sp.Name, projectSlug, platform)
		if err != nil {
			return r.setFailed(ctx, &sp, fmt.Sprintf("create project: %v", err))
		}
		logger.Info("project created", "slug", project.Slug)
	}

	secretData, keyStatuses, err := reconcileKeys(ctx, r.SentryClient, org, project.Slug, sp.Spec.Keys, sp.Spec.DefaultRateLimit, sp.Status.Keys, true)
	if err != nil {
		return r.setFailed(ctx, &sp, fmt.Sprintf("reconcile keys: %v", err))
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: sp.Namespace,
		},
	}
	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.StringData = secretData
		setManagedLabels(secret)
		return controllerutil.SetControllerReference(&sp, secret, r.Scheme)
	})
	if err != nil {
		return r.setFailed(ctx, &sp, fmt.Sprintf("sync secret: %v", err))
	}
	logger.Info("secret synced", "name", secretName, "result", result)

	now := metav1.Now()
	sp.Status.ProjectSlug = project.Slug
	sp.Status.SecretName = secretName
	sp.Status.Keys = keyStatuses
	sp.Status.LastSyncTime = &now
	sp.Status.ObservedGeneration = sp.Generation
	setCondition(&sp.Status.Conditions, sp.Generation, sentryv1alpha1.ConditionReady, metav1.ConditionTrue,
		sentryv1alpha1.ReasonProjectProvisioned, "Sentry project provisioned and secret synced")

	if err := r.Status().Update(ctx, &sp); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: r.Config.RequeueInterval}, nil
}

func (r *SentryProjectReconciler) handleDeletion(ctx context.Context, sp *sentryv1alpha1.SentryProject) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(sp, finalizerName) {
		return ctrl.Result{}, nil
	}

	retain := r.Config.DefaultRetainOnDelete
	if sp.Spec.RetainOnDelete != nil {
		retain = *sp.Spec.RetainOnDelete
	}

	if !retain {
		org := coalesce(sp.Spec.Organization, r.Config.DefaultOrganization)
		slug := coalesce(sp.Spec.ProjectSlug, sp.Name)

		if org != "" && slug != "" {
			logger.Info("deleting Sentry project", "org", org, "slug", slug)
			if err := r.SentryClient.DeleteProject(ctx, org, slug); err != nil {
				return ctrl.Result{}, fmt.Errorf("delete sentry project: %w", err)
			}
		}
	} else {
		logger.Info("retaining Sentry project on delete (retainOnDelete=true)")
	}

	secretName := coalesce(sp.Spec.SecretName, sp.Name+"-sentry")
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: sp.Namespace}, secret)
	if err == nil {
		if err := r.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("delete secret: %w", err)
		}
	}

	controllerutil.RemoveFinalizer(sp, finalizerName)
	if err := r.Update(ctx, sp); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *SentryProjectReconciler) setFailed(ctx context.Context, sp *sentryv1alpha1.SentryProject, msg string) (ctrl.Result, error) {
	log.FromContext(ctx).Error(errors.New(msg), "reconcile failed")
	setCondition(&sp.Status.Conditions, sp.Generation, sentryv1alpha1.ConditionReady, metav1.ConditionFalse,
		sentryv1alpha1.ReasonProvisionFailed, msg)
	_ = r.Status().Update(ctx, sp)
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *SentryProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sentryv1alpha1.SentryProject{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// setCondition upserts a condition into a conditions slice.
func setCondition(conditions *[]metav1.Condition, generation int64, condType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	for i, c := range *conditions {
		if c.Type == condType {
			if c.Status != status || c.Reason != reason {
				(*conditions)[i].Status = status
				(*conditions)[i].Reason = reason
				(*conditions)[i].Message = message
				(*conditions)[i].LastTransitionTime = now
				(*conditions)[i].ObservedGeneration = generation
			}
			return
		}
	}
	*conditions = append(*conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
		ObservedGeneration: generation,
	})
}

// setManagedLabels ensures the Secret is labelled as managed by the operator.
func setManagedLabels(secret *corev1.Secret) {
	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}
	secret.Labels["app.kubernetes.io/managed-by"] = "sentry-operator"
}

func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
