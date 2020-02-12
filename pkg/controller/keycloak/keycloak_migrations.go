package keycloak

import (
	"fmt"
	"reflect"
	"time"

	"github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/keycloak/keycloak-operator/pkg/common"
	"github.com/keycloak/keycloak-operator/pkg/model"
	v13 "k8s.io/api/apps/v1"
)

type Migrator interface {
	Migrate(cr *v1alpha1.Keycloak, currentState *common.ClusterState, desiredState common.DesiredClusterState) (common.DesiredClusterState, error)
}

type DefaultMigrator struct {
}

func NewDefaultMigrator() *DefaultMigrator {
	return &DefaultMigrator{}
}

func (i *DefaultMigrator) Migrate(cr *v1alpha1.Keycloak, currentState *common.ClusterState, desiredState common.DesiredClusterState) (common.DesiredClusterState, error) {
	if needsMigration(cr, currentState) {
		desiredImage := model.KeycloakImage
		if cr.Spec.ImageOverrides.Keycloak != "" {
			desiredImage = cr.Spec.ImageOverrides.Keycloak
		}

		if cr.Spec.Profile == common.RHSSOProfile {
			desiredImage = model.RHSSOImage
			if cr.Spec.ImageOverrides.RHSSO != "" {
				desiredImage = cr.Spec.ImageOverrides.RHSSO
			}
		}

		log.Info(fmt.Sprintf("Performing migration from '%s' to '%s'", currentState.KeycloakDeployment.Spec.Template.Spec.Containers[0].Image, desiredImage))
		deployment := findDeployment(desiredState)
		if deployment != nil {
			log.Info("Number of replicas decreased to 1")
			deployment.Spec.Replicas = &[]int32{1}[0]
		}

		if cr.Spec.Migrations.Backups.Enabled == true {
			// Should use current Postgresql Image to do the OneTimeLocalBackup
			if postgresqlNeedsMigration(cr, currentState) {
				cr.Spec.ImageOverrides.Postgresql = currentState.PostgresqlDeployment.Spec.Template.Spec.Containers[0].Image
			}
			desiredState = oneTimeLocalDatabaseBackup(cr, desiredState)
		}
	} else if postgresqlNeedsMigration(cr, currentState) {
		if cr.Spec.Migrations.Backups.Enabled == true {
			cr.Spec.ImageOverrides.Postgresql = currentState.PostgresqlDeployment.Spec.Template.Spec.Containers[0].Image
			desiredState = oneTimeLocalDatabaseBackup(cr, desiredState)
		}
	}

	return desiredState, nil
}

func needsMigration(cr *v1alpha1.Keycloak, currentState *common.ClusterState) bool {
	if currentState.KeycloakDeployment == nil {
		return false
	}

	currentImage := currentState.KeycloakDeployment.Spec.Template.Spec.Containers[0].Image
	desiredImage := model.KeycloakImage
	if cr.Spec.ImageOverrides.Keycloak != "" {
		desiredImage = cr.Spec.ImageOverrides.Keycloak
	}

	if cr.Spec.Profile == common.RHSSOProfile {
		desiredImage = model.RHSSOImage
		if cr.Spec.ImageOverrides.RHSSO != "" {
			desiredImage = cr.Spec.ImageOverrides.RHSSO
		}
	}
	return desiredImage != currentImage
}

func findDeployment(desiredState common.DesiredClusterState) *v13.StatefulSet {
	for _, v := range desiredState {
		if (reflect.TypeOf(v) == reflect.TypeOf(common.GenericUpdateAction{})) {
			updateAction := v.(common.GenericUpdateAction)
			if (reflect.TypeOf(updateAction.Ref) == reflect.TypeOf(&v13.StatefulSet{})) {
				statefulSet := updateAction.Ref.(*v13.StatefulSet)
				if statefulSet.ObjectMeta.Name == model.KeycloakDeploymentName {
					return statefulSet
				}
			}
		}
	}
	return nil
}

func postgresqlNeedsMigration(cr *v1alpha1.Keycloak, currentState *common.ClusterState) bool {
	if currentState.PostgresqlDeployment == nil {
		return false
	}

	currentImage := currentState.PostgresqlDeployment.Spec.Template.Spec.Containers[0].Image
	desiredImage := model.PostgresqlImage
	if cr.Spec.ImageOverrides.Postgresql != "" {
		desiredImage = cr.Spec.ImageOverrides.Postgresql
	}

	return desiredImage != currentImage
}

// oneTimeLocalDatabaseBackup backups database by OneTimeLocalBackup
func oneTimeLocalDatabaseBackup(cr *v1alpha1.Keycloak, desiredState common.DesiredClusterState) common.DesiredClusterState {
	backupCr := &v1alpha1.KeycloakBackup{}
	backupCr.Namespace = cr.Namespace
	backupCr.Name = "migration-backup-" + time.Now().Format("20060102-150405")

	backupAction := common.GenericCreateAction{
		Ref: model.PostgresqlBackup(backupCr, cr),
		Msg: "Create Local Backup job",
	}
	volumeClaimAction := common.GenericCreateAction{
		Ref: model.PostgresqlBackupPersistentVolumeClaim(backupCr),
		Msg: "Create Local Backup Persistent Volume Claim",
	}

	log.Info(fmt.Sprintf("Ready to perform migration OnetimeLocalBackup with name '%s'", backupCr.Name))
	desiredState = desiredState.AddAction(backupAction)
	desiredState = desiredState.AddAction(volumeClaimAction)

	return desiredState
}
