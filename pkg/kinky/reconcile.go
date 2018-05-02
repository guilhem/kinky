package kinky

import (
	"fmt"

	api "github.com/barpilot/kinky/pkg/apis/kinky/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	"github.com/sirupsen/logrus"
)

// Reconcile reconciles the kinky cluster's state to the spec specified by cl
// by preparing the TLS secrets, deploying the etcd and kinky cluster,
// and finally updating the vault deployment if needed.
func Reconcile(cl *api.Cluster) (err error) {
	cl = cl.DeepCopy()
	// Simulate initializer.
	changed := cl.SetDefaults()
	if changed {
		return action.Update(cl)
	}
	// After first time reconcile, phase will switch to "Running".
	if cl.Status.Phase == api.ClusterPhaseInitial {
		err = prepareEtcdTLSSecrets(cl)
		if err != nil {
			return err
		}
		// etcd cluster should only be created in first time reconcile.
		ec, err := deployEtcdCluster(cl)
		if err != nil {
			return err
		}
		// Check if etcd cluster is up and running.
		// If not, we need to wait until etcd cluster is up before proceeding to the next state;
		// Hence, we return from here and let the Watch triggers the handler again.
		ready, err := isEtcdClusterReady(ec)
		if err != nil {
			return fmt.Errorf("failed to check if etcd cluster is ready: %v", err)
		}
		if !ready {
			logrus.Infof("Waiting for EtcdCluster (%v) to become ready", ec.Name)
			return nil
		}
	}

	// err = prepareDefaultVaultTLSSecrets(cl)
	// if err != nil {
	// 	return err
	// }
	//
	// err = prepareVaultConfig(cl)
	// if err != nil {
	// 	return err
	// }
	//
	// err = deployVault(cl)
	// if err != nil {
	// 	return err
	// }
	//
	// err = syncVaultClusterSize(cl)
	// if err != nil {
	// 	return err
	// }
	//
	// vcs, err := getVaultStatus(cl)
	// if err != nil {
	// 	return err
	// }
	//
	// err = syncUpgrade(cl, vcs)
	// if err != nil {
	// 	return err
	// }
	//
	// return updateVaultStatus(cl, vcs)
	return nil
}
