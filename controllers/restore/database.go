package pulp_restore

import (
	"context"
	"time"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// restoreDatabaseData scales down the pods and runs a pg_restore
func (r *PulpRestoreReconciler) restoreDatabaseData(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, backupDir string, pod *corev1.Pod) error {
	log := log.FromContext(ctx)
	backupFile := "pulp.db"

	//[TODO] fix this
	// WORKAROUND!!!! Giving some time to pulp CR be created
	// sometimes the scale down process was failing with kube-api returning an error
	// because the object has been modified and asking to try again
	// I think one of the reasons that it is happening is because pulp CR
	// was in the middle of its "creation process" and this kludge did a "relief"
	time.Sleep(5 * time.Second)

	// retrieve pg credentials and address
	pgConfig := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Status.PostgresSecret, Namespace: pulpRestore.Namespace}, pgConfig); err != nil {
		log.Error(err, "Failed to find postgres-configuration secret")
		return err
	}

	// wait until database pod is ready
	log.Info("Waiting db pod get into a READY state ...")
	r.waitDBReady(ctx, pulpRestore.Namespace, pulpRestore.Spec.DeploymentName+"-database")

	// [TODO] we need to define when to run pg_restore
	// in the "first" execution everything goes fine
	// during a reconcile loop (or, for example, if the operator pod gets reprovisioned) the pg_restore will fail
	// Here is an error from a subsequent execution of pg_restore:
	/* pg_restore: from TOC entry 4583; 2606 18967 FK CONSTRAINT rpm_variant rpm_variant_repository_id_621855c9_fk_core_repository_pulp_id pulp
	pg_restore: error: could not execute query: ERROR:  constraint "rpm_variant_repository_id_621855c9_fk_core_repository_pulp_id" for relation "rpm_variant" already exists
	Command was: ALTER TABLE ONLY public.rpm_variant
	    ADD CONSTRAINT rpm_variant_repository_id_621855c9_fk_core_repository_pulp_id FOREIGN KEY (repository_id) REFERENCES public.core_repository(pulp_id) DEFERRABLE INITIALLY DEFERRED;
	pg_restore: warning: errors ignored on restore: 843 */

	// run pg_restore
	execCmd := []string{
		"bash", "-c", "cat" + backupDir + "/" + backupFile + "| PGPASSWORD=" + string(pgConfig.Data["password"]),
		"psql -U " + string(pgConfig.Data["username"]),
		"-h " + string(pgConfig.Data["host"]),
		"-U " + string(pgConfig.Data["username"]),
		"-d " + string(pgConfig.Data["database"]),
		"-p " + string(pgConfig.Data["port"]),
	}

	log.Info("Running db restore ...")
	if _, err := r.containerExec(pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace); err != nil {
		log.Error(err, "Failed to restore postgres data")
		return err
	}

	log.Info("Database restore finished!")

	return nil
}

// waitDBReady waits until db container gets into a "READY" state
func (r *PulpRestoreReconciler) waitDBReady(ctx context.Context, namespace, stsName string) error {
	var err error
	for timeout := 0; timeout < 120; timeout++ {
		sts := &appsv1.StatefulSet{}
		err = r.Get(ctx, types.NamespacedName{Name: stsName, Namespace: namespace}, sts)
		if sts.Status.ReadyReplicas == sts.Status.Replicas {
			return nil
		}
		time.Sleep(time.Second)
	}
	return err
}
