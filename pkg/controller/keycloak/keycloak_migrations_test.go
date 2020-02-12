package keycloak

import (
	"testing"

	"github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/keycloak/keycloak-operator/pkg/common"
	"github.com/keycloak/keycloak-operator/pkg/model"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
)

func TestKeycloakMigrations_Test_No_Need_For_Migration_On_Empty_Desired_State(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{}
	currentState := common.ClusterState{}
	desiredState := common.DesiredClusterState{}

	// when
	migratedActions, error := migrator.Migrate(cr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.Equal(t, desiredState, migratedActions)
}

func TestKeycloakMigrations_Test_No_Need_For_Migration_On_Missing_Deployment_In_Desired_State(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{}

	keycloakDeployment := model.KeycloakDeployment(cr)
	keycloakDeployment.Spec.Replicas = &[]int32{5}[0]
	keycloakDeployment.Spec.Template.Spec.Containers[0].Image = "old_image" //nolint

	currentState := common.ClusterState{
		KeycloakDeployment: keycloakDeployment,
	}

	desiredState := common.DesiredClusterState{}
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: model.KeycloakService(cr),
	})

	// when
	migratedActions, error := migrator.Migrate(cr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.Equal(t, desiredState, migratedActions)
}

func TestKeycloakMigrations_Test_Migrating_Image(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{}

	keycloakDeployment := model.KeycloakDeployment(cr)
	keycloakDeployment.Spec.Replicas = &[]int32{5}[0]
	keycloakDeployment.Spec.Template.Spec.Containers[0].Image = "old_image" //nolint

	currentState := common.ClusterState{
		KeycloakDeployment: keycloakDeployment,
	}

	desiredState := common.DesiredClusterState{}
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: model.KeycloakDeployment(cr),
	})

	// when
	migratedActions, error := migrator.Migrate(cr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.Equal(t, int32(1), *migratedActions[0].(common.GenericUpdateAction).Ref.(*v1.StatefulSet).Spec.Replicas)
}

func TestKeycloakMigrations_Test_Migrating_RHSSO_Image(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			Profile: common.RHSSOProfile,
		},
	}

	keycloakDeployment := model.RHSSODeployment(cr)
	keycloakDeployment.Spec.Replicas = &[]int32{5}[0]
	keycloakDeployment.Spec.Template.Spec.Containers[0].Image = "old_image" //nolint

	currentState := common.ClusterState{
		KeycloakDeployment: keycloakDeployment,
	}

	desiredState := common.DesiredClusterState{}
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: model.RHSSODeployment(cr),
	})

	// when
	migratedActions, error := migrator.Migrate(cr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.Equal(t, int32(1), *migratedActions[0].(common.GenericUpdateAction).Ref.(*v1.StatefulSet).Spec.Replicas)
}

func TestKeycloakMigrations_Test_No_Backup_Without_Migration(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			Migrations: v1alpha1.MigrateConfig{
				Backups: v1alpha1.BackupConfig{
					Enabled: true,
				},
			},
		},
	}

	keycloakDeployment := model.KeycloakDeployment(cr)
	currentState := common.ClusterState{
		KeycloakDeployment: keycloakDeployment,
	}
	desiredState := common.DesiredClusterState{}
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: keycloakDeployment,
	})

	// when
	migratedActions, error := migrator.Migrate(cr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.Equal(t, desiredState, migratedActions)
}

func TestKeycloakMigrations_Test_Backup_Happen_With_Enabled_BackupConfig_And_Overrided_Keycloak_Image(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{}
	migrateCr := &v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			ImageOverrides: v1alpha1.KeycloakRelatedImages{
				Keycloak: "keycloak:1.0.0",
			},
			Migrations: v1alpha1.MigrateConfig{
				Backups: v1alpha1.BackupConfig{
					Enabled: true,
				},
			},
		},
	}

	keycloakDeployment := model.KeycloakDeployment(cr)
	currentState := common.ClusterState{
		KeycloakDeployment: keycloakDeployment,
	}
	desiredState := common.DesiredClusterState{}
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: keycloakDeployment,
	})

	// when
	migratedActions, error := migrator.Migrate(migrateCr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.NotEqual(t, desiredState, migratedActions)
}

func TestKeycloakMigrations_Test_Backup_Happen_With_Enabled_BackupConfig_And_Overrided_Postgresql_Image(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{}
	migrateCr := &v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			ImageOverrides: v1alpha1.KeycloakRelatedImages{
				Postgresql: "postgresql:1.0.0",
			},
			Migrations: v1alpha1.MigrateConfig{
				Backups: v1alpha1.BackupConfig{
					Enabled: true,
				},
			},
		},
	}

	postgresqlDeployment := model.PostgresqlDeployment(cr)
	currentState := common.ClusterState{
		PostgresqlDeployment: postgresqlDeployment,
	}
	desiredState := common.DesiredClusterState{}
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: postgresqlDeployment,
	})

	// when
	migratedActions, error := migrator.Migrate(migrateCr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.NotEqual(t, desiredState, migratedActions)
}

func TestKeycloakMigrations_Test_Backup_Happen_With_Enabled_BackupConfig_And_Two_Overrided_Images(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{}
	migrateCr := &v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			ImageOverrides: v1alpha1.KeycloakRelatedImages{
				Keycloak:   "keycloak:1.0.0",
				Postgresql: "postgresql:1.0.0",
			},
			Migrations: v1alpha1.MigrateConfig{
				Backups: v1alpha1.BackupConfig{
					Enabled: true,
				},
			},
		},
	}

	keycloakDeployment := model.KeycloakDeployment(cr)
	postgresqlDeployment := model.PostgresqlDeployment(cr)
	currentState := common.ClusterState{
		KeycloakDeployment:   keycloakDeployment,
		PostgresqlDeployment: postgresqlDeployment,
	}
	desiredState := common.DesiredClusterState{}
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: keycloakDeployment,
	})
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: postgresqlDeployment,
	})

	// when
	migratedActions, error := migrator.Migrate(migrateCr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.NotEqual(t, desiredState, migratedActions)
}

func TestKeycloakMigrations_Test_No_Need_Backup_Without_Enabled_BackupConfig(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{}
	migrateCr := &v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			ImageOverrides: v1alpha1.KeycloakRelatedImages{
				Keycloak: "keycloak:1.0.0",
			},
			Migrations: v1alpha1.MigrateConfig{
				Backups: v1alpha1.BackupConfig{
					Enabled: false,
				},
			},
		},
	}

	keycloakDeployment := model.KeycloakDeployment(cr)
	currentState := common.ClusterState{
		KeycloakDeployment: keycloakDeployment,
	}
	desiredState := common.DesiredClusterState{}
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: keycloakDeployment,
	})

	// when
	migratedActions, error := migrator.Migrate(migrateCr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.Equal(t, desiredState, migratedActions)
}

func TestKeycloakMigrations_Test_No_Need_Backup_Without_Migrations_Property(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{}
	migrateCr := &v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			ImageOverrides: v1alpha1.KeycloakRelatedImages{
				Keycloak: "keycloak:1.0.0",
			},
		},
	}

	keycloakDeployment := model.KeycloakDeployment(cr)
	currentState := common.ClusterState{
		KeycloakDeployment: keycloakDeployment,
	}
	desiredState := common.DesiredClusterState{}
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: keycloakDeployment,
	})

	// when
	migratedActions, error := migrator.Migrate(migrateCr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.Equal(t, desiredState, migratedActions)
}

func TestKeycloakMigrations_Test_No_Need_Backup_Without_Overrided_Images(t *testing.T) {
	// given
	migrator := NewDefaultMigrator()
	cr := &v1alpha1.Keycloak{}
	migrateCr := &v1alpha1.Keycloak{
		Spec: v1alpha1.KeycloakSpec{
			Migrations: v1alpha1.MigrateConfig{
				Backups: v1alpha1.BackupConfig{
					Enabled: true,
				},
			},
		},
	}

	keycloakDeployment := model.KeycloakDeployment(cr)
	currentState := common.ClusterState{
		KeycloakDeployment: keycloakDeployment,
	}
	desiredState := common.DesiredClusterState{}
	desiredState = append(desiredState, common.GenericUpdateAction{
		Ref: keycloakDeployment,
	})

	// when
	migratedActions, error := migrator.Migrate(migrateCr, &currentState, desiredState)

	// then
	assert.Nil(t, error)
	assert.Equal(t, desiredState, migratedActions)
}
