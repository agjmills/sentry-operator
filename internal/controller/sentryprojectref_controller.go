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

// SentryProjectRefReconciler reconciles SentryProjectRef objects.
type SentryProjectRefReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	SentryClient *sentry.Client
	Config       Config
}

// +kubebuilder:rbac:groups=sentry-operator.io,resources=sentryprojectrefs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sentry-operator.io,resources=sentryprojectrefs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sentry-operator.io,resources=sentryprojectrefs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *SentryProjectRefReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ref sentryv1alpha1.SentryProjectRef
	if err := r.Get(ctx, req.NamespacedName, &ref); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !ref.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &ref)
	}

	if !controllerutil.ContainsFinalizer(&ref, finalizerName) {
		controllerutil.AddFinalizer(&ref, finalizerName)
		if err := r.Update(ctx, &ref); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	org := coalesce(ref.Spec.Organization, r.Config.DefaultOrganization)
	secretName := coalesce(ref.Spec.SecretName, ref.Name+"-sentry")

	if org == "" {
		return r.setFailed(ctx, &ref, "organization must be set (via spec or operator default)")
	}

	logger.Info("reconciling SentryProjectRef", "org", org, "slug", ref.Spec.ProjectSlug)

	project, err := r.SentryClient.GetProject(ctx, org, ref.Spec.ProjectSlug)
	if err != nil {
		return r.setFailed(ctx, &ref, fmt.Sprintf("get project: %v", err))
	}
	if project == nil {
		return r.setFailed(ctx, &ref, fmt.Sprintf("project %q not found in org %q — SentryProjectRef only references existing projects", ref.Spec.ProjectSlug, org))
	}

	// createMissing=false: we only read existing keys, never create them.
	secretData, err := reconcileKeys(ctx, r.SentryClient, org, project.Slug, ref.Spec.Keys, ref.Spec.DefaultRateLimit, false)
	if err != nil {
		return r.setFailed(ctx, &ref, fmt.Sprintf("reconcile keys: %v", err))
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ref.Namespace,
		},
	}
	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.StringData = secretData
		return controllerutil.SetControllerReference(&ref, secret, r.Scheme)
	})
	if err != nil {
		return r.setFailed(ctx, &ref, fmt.Sprintf("sync secret: %v", err))
	}
	logger.Info("secret synced", "name", secretName, "result", result)

	now := metav1.Now()
	ref.Status.ProjectSlug = project.Slug
	ref.Status.SecretName = secretName
	ref.Status.LastSyncTime = &now
	ref.Status.ObservedGeneration = ref.Generation
	setCondition(&ref.Status.Conditions, ref.Generation, sentryv1alpha1.ConditionReady, metav1.ConditionTrue,
		sentryv1alpha1.ReasonProjectProvisioned, "Sentry project found and secret synced")

	if err := r.Status().Update(ctx, &ref); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: r.Config.RequeueInterval}, nil
}

func (r *SentryProjectRefReconciler) handleDeletion(ctx context.Context, ref *sentryv1alpha1.SentryProjectRef) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(ref, finalizerName) {
		return ctrl.Result{}, nil
	}

	// Never touch the Sentry project — only clean up the Secret.
	secretName := coalesce(ref.Spec.SecretName, ref.Name+"-sentry")
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ref.Namespace}, secret)
	if err == nil {
		if err := r.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("delete secret: %w", err)
		}
	}

	controllerutil.RemoveFinalizer(ref, finalizerName)
	if err := r.Update(ctx, ref); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *SentryProjectRefReconciler) setFailed(ctx context.Context, ref *sentryv1alpha1.SentryProjectRef, msg string) (ctrl.Result, error) {
	log.FromContext(ctx).Error(errors.New(msg), "reconcile failed")
	setCondition(&ref.Status.Conditions, ref.Generation, sentryv1alpha1.ConditionReady, metav1.ConditionFalse,
		sentryv1alpha1.ReasonProvisionFailed, msg)
	_ = r.Status().Update(ctx, ref)
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *SentryProjectRefReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sentryv1alpha1.SentryProjectRef{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
