package landscaper

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/helm/pkg/helm"
)

// Executor is responsible for applying a desired landscape to the actual landscape
type Executor interface {
	Apply(Components, Components) error

	CreateComponent(*Component) error
	UpdateComponent(*Component) error
	DeleteComponent(*Component) error
}

type executor struct {
	helmClient     helm.Interface
	chartLoader    ChartLoader
	kubeSecrets    SecretsWriteDeleter
	dryRun         bool
	wait           bool
	waitTimeout    int64
	disabledStages []string
}

// NewExecutor is a factory method to create a new Executor
func NewExecutor(helmClient helm.Interface, chartLoader ChartLoader, kubeSecrets SecretsWriteDeleter, dryRun bool, wait bool, waitTimeout int64, disabledStages []string) Executor {
	return &executor{
		helmClient:     helmClient,
		chartLoader:    chartLoader,
		kubeSecrets:    kubeSecrets,
		dryRun:         dryRun,
		wait:           wait,
		waitTimeout:    waitTimeout,
		disabledStages: disabledStages,
	}
}

// gatherForcedUpdates returns a map that for each to-be-updated component indicates if it needs a forced update.
// there may be several reasons to do so: releases that differ only in secret values are forced so that pods will restart with the new values; releases that differ in namespace cannot be updated
func (e *executor) gatherForcedUpdates(current, update Components) (map[string]bool, error) {
	needForcedUpdate := map[string]bool{}

	for _, cmp := range update {
		// releases that differ only in secret values are forced so that pods will restart with the new values
		for _, curCmp := range current {
			if curCmp.Name == cmp.Name && isOnlySecretValueDiff(*curCmp, *cmp) {
				logrus.Infof("%s differs in secrets values only; don't update but delete + create instead", cmp.Name)
				needForcedUpdate[cmp.Name] = true
			}
		}
		if curCmp := current[cmp.Name]; curCmp != nil {
			if curCmp.Namespace != cmp.Namespace {
				logrus.Infof("%s differs in namespace; don't update but delete + create instead", cmp.Name)
				needForcedUpdate[cmp.Name] = true
			}
		}
	}

	return needForcedUpdate, nil
}

// Apply transforms the current state into the desired state
func (e *executor) Apply(desired, current Components) error {
	create, update, delete := diff(desired, current)

	// some to-be-updated components need a delete + create instead
	needForcedUpdate, err := e.gatherForcedUpdates(current, update)
	if err != nil {
		return err
	}

	// delete+create pairs will never work in dry run since the dry-run "deleted" component will exist in create
	if !e.dryRun {
		create, update, delete = integrateForcedUpdates(current, create, update, delete, needForcedUpdate)
	}

	logrus.WithFields(logrus.Fields{"create": len(create), "update": len(update), "delete": len(delete)}).Info("Apply desired state")

	for _, cmp := range delete {
		_, cmpForcedUpdate := needForcedUpdate[cmp.Name]
		if e.stageEnabled("delete") || (e.stageEnabled("update") && cmpForcedUpdate) {
			logrus.Infof("Delete: %s", cmp.Name)
			if err := e.DeleteComponent(cmp); err != nil {
				logrus.WithFields(logrus.Fields{"error": err, "component": cmp}).Error("DeleteComponent failed")
				return err
			}
		}
	}

	if e.stageEnabled("update") {
		for _, cmp := range update {
			if err := logDifferences(logrus.Infof, "Update: "+cmp.Name, current[cmp.Name], cmp); err != nil {
				return err
			}
			if err := e.UpdateComponent(cmp); err != nil {
				logrus.WithFields(logrus.Fields{"error": err, "component": cmp}).Error("UpdateComponent failed")
				return err
			}
		}
	}

	for _, cmp := range create {
		_, cmpForcedUpdate := needForcedUpdate[cmp.Name]
		if e.stageEnabled("create") || (e.stageEnabled("update") && cmpForcedUpdate) {
			if err := logDifferences(logrus.Infof, "Create: "+cmp.Name, nil, cmp); err != nil {
				return err
			}

			if err := e.CreateComponent(cmp); err != nil {
				logrus.WithFields(logrus.Fields{"error": err, "component": cmp}).Error("CreateComponent failed")
				return err
			}
		}
	}

	logrus.WithFields(logrus.Fields{"created": len(create), "updated": len(update), "deleted": len(delete)}).Info("Applied desired state successfully")
	return nil
}

func (e *executor) stageEnabled(stage string) bool {
	for _, stageDisabled := range e.disabledStages {
		if stageDisabled == stage {
			return false
		}
	}
	return true
}

// CreateComponent creates the given Component
func (e *executor) CreateComponent(cmp *Component) error {
	// We need to ensure the chart is available on the local system. LoadChart will ensure
	// this is the case by downloading the chart if it is not there yet
	chartRef, err := cmp.FullChartRef()
	if err != nil {
		return err
	}
	_, chartPath, err := e.chartLoader.Load(chartRef)
	if err != nil {
		return err
	}

	rawValues, err := cmp.Configuration.YAML()
	if err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"release":   cmp.Name,
		"chart":     cmp.Release.Chart,
		"chartPath": chartPath,
		"rawValues": rawValues,
		"values":    cmp.Configuration,
		"dryrun":    e.dryRun,
	}).Debug("Create component")

	if len(cmp.SecretValues) > 0 && !e.dryRun {
		err = e.kubeSecrets.Write(cmp.Name, cmp.Namespace, cmp.SecretValues)
		if err != nil {
			return err
		}
	}

	_, err = e.helmClient.InstallRelease(
		chartPath,
		cmp.Namespace,
		helm.ValueOverrides([]byte(rawValues)),
		helm.ReleaseName(cmp.Name),
		helm.InstallDryRun(e.dryRun),
		helm.InstallReuseName(true),
		helm.InstallWait(e.wait),
		helm.InstallTimeout(e.waitTimeout),
	)
	if err != nil {
		return errors.New(grpc.ErrorDesc(err))
	}

	return nil
}

// UpdateComponent updates the given Component
func (e *executor) UpdateComponent(cmp *Component) error {
	// We need to ensure the chart is available on the local system. LoadChart will ensure
	// this is the case by downloading the chart if it is not there yet
	chartRef, err := cmp.FullChartRef()
	if err != nil {
		return err
	}
	_, chartPath, err := e.chartLoader.Load(chartRef)
	if err != nil {
		return err
	}

	rawValues, err := cmp.Configuration.YAML()
	if err != nil {
		return err
	}

	if !e.dryRun {
		if e.stageEnabled("deleteSecrets") || len(cmp.SecretValues) > 0 {
			err = e.kubeSecrets.Delete(cmp.Name, cmp.Namespace)
		}

		if len(cmp.SecretValues) > 0 {
			err = e.kubeSecrets.Write(cmp.Name, cmp.Namespace, cmp.SecretValues)
			if err != nil {
				return err
			}
		}
	}

	logrus.WithFields(logrus.Fields{
		"release":   cmp.Name,
		"chart":     cmp.Release.Chart,
		"chartPath": chartPath,
		"values":    cmp.Configuration,
		"dryrun":    e.dryRun,
	}).Debug("Update component")

	_, err = e.helmClient.UpdateRelease(
		cmp.Name,
		chartPath,
		helm.UpdateValueOverrides([]byte(rawValues)),
		helm.UpgradeDryRun(e.dryRun),
		helm.UpgradeWait(e.wait),
		helm.UpgradeTimeout(e.waitTimeout),
	)
	if err != nil {
		return errors.New(grpc.ErrorDesc(err))
	}

	return nil
}

// DeleteComponent removes the given Component
func (e *executor) DeleteComponent(cmp *Component) error {
	logrus.WithFields(logrus.Fields{
		"release": cmp.Name,
		"values":  cmp.Configuration,
		"dryrun":  e.dryRun,
	}).Debug("Delete component")

	if len(cmp.SecretValues) > 0 && !e.dryRun {
		err := e.kubeSecrets.Delete(cmp.Name, cmp.Namespace)
		if err != nil {
			return err
		}
	}

	if !e.dryRun {
		_, err := e.helmClient.DeleteRelease(
			cmp.Name,
			helm.DeletePurge(true),
			helm.DeleteDryRun(e.dryRun),
		)
		if err != nil {
			return errors.New(grpc.ErrorDesc(err))
		}
	}

	return nil
}

// diff takes desired and current components, and returns the components to create, update and delete to get from current to desired
func diff(desired, current Components) (Components, Components, Components) {
	create := Components{}
	update := Components{}
	delete := Components{}

	for name, desiredCmp := range desired {
		if currentCmp, ok := current[name]; ok {
			if !desiredCmp.Equals(currentCmp) {
				update[name] = desiredCmp
			}
		} else {
			create[name] = desiredCmp
		}
	}

	for name, currentCmp := range current {
		if _, ok := desired[name]; !ok {
			delete[name] = currentCmp
		}
	}

	return create, update, delete
}

// componentDiffText returns a diff as text. current and desired can be nil and indicate non-existence (e.g. current nil and desired non-nil means: create)
func componentDiffText(current, desired *Component) (string, error) {
	cText, dText := []string{}, []string{}
	cName, dName := "<none>", "<none>"
	if current != nil {
		cs, err := json.MarshalIndent(current, "", "  ")
		if err != nil {
			return "", err
		}
		cText = difflib.SplitLines(string(cs))
		cName = current.Name
	}
	if desired != nil {
		ds, err := json.MarshalIndent(desired, "", "  ")
		if err != nil {
			return "", err
		}
		dText = difflib.SplitLines(string(ds))
		dName = desired.Name
	}

	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        cText,
		FromFile: "Current " + cName,
		B:        dText,
		ToFile:   "Desired " + dName,
		Context:  3,
	})
}

func logDifferences(logf func(format string, args ...interface{}), action string, current, desired *Component) error {
	diff, err := componentDiffText(current, desired)
	if err != nil {
		return err
	}
	logf("%s", action)
	if diff != "" {
		logf("Diff:\n%s", diff)
	}
	if current != nil && desired != nil && !reflect.DeepEqual(current.SecretValues, desired.SecretValues) {
		logrus.Info("Diff: secrets have changed, not shown here")
	}
	return nil
}

// integrateForcedUpdates removes forceUpdate from update and inserts it into delete + create
func integrateForcedUpdates(current, create, update, delete Components, forceUpdate map[string]bool) (Components, Components, Components) {
	fixUpdate := Components{}
	for _, cmp := range update {
		if forceUpdate[cmp.Name] {
			if currentCmp, ok := current[cmp.Name]; ok {
				delete[currentCmp.Name] = currentCmp // delete the current component
			}
			create[cmp.Name] = cmp // create cmp, by definition a desired component
		} else {
			fixUpdate[cmp.Name] = cmp
		}
	}
	return create, fixUpdate, delete
}

// isOnlySecretValueDiff tells whether the given Components differ in their .SecretValues fields and are identical otherwise
func isOnlySecretValueDiff(a, b Component) bool {
	secValsEqual := reflect.DeepEqual(a.SecretValues, b.SecretValues)
	a.SecretValues = SecretValues{}
	b.SecretValues = SecretValues{}
	return !secValsEqual && reflect.DeepEqual(a, b)
}
