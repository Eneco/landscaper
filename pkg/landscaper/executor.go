package landscaper

import (
	"encoding/json"
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/pmezard/go-difflib/difflib"
	"google.golang.org/grpc"
	"k8s.io/helm/pkg/helm"
)

// Executor is responsible for applying a desired landscape to the actual landscape
type Executor interface {
	Apply([]*Component, []*Component) error

	CreateComponent(*Component) error
	UpdateComponent(*Component) error
	DeleteComponent(*Component) error
}

type executor struct {
	env             *Environment
	secretsProvider SecretsProvider
}

// NewExecutor is a factory method to create a new Executor
func NewExecutor(env *Environment, secretsProvider SecretsProvider) Executor {
	return &executor{
		env:             env,
		secretsProvider: secretsProvider,
	}
}

// Apply transforms the current state into the desired state
func (e *executor) Apply(desired, current []*Component) error {
	create, update, delete := diff(desired, current)

	logrus.WithFields(logrus.Fields{"create": len(create), "update": len(update), "delete": len(delete)}).Info("Apply desired state")

	if err := logDifferences(current, create, update, delete, logrus.Infof); err != nil {
		return err
	}

	for _, cmp := range delete {
		if err := e.DeleteComponent(cmp); err != nil {
			logrus.Error("DeleteComponent failed", err)
			return err
		}
	}

	for _, cmp := range create {
		if err := e.CreateComponent(cmp); err != nil {
			logrus.Error("CreateComponent failed", err)
			return err
		}
	}

	for _, cmp := range update {
		if err := e.UpdateComponent(cmp); err != nil {
			logrus.Error("UpdateComponent failed", err)
			return err
		}
	}

	logrus.WithFields(logrus.Fields{"created": len(create), "updated": len(update), "deleted": len(delete)}).Info("Applied desired state sucessfully")
	return nil
}

// CreateComponent creates the given Component
func (e *executor) CreateComponent(cmp *Component) error {
	// We need to ensure the chart is available on the local system. LoadChart will ensure
	// this is the case by downloading the chart if it is not there yet
	chartRef, err := cmp.FullChartRef()
	if err != nil {
		return err
	}
	_, chartPath, err := e.env.ChartLoader.Load(chartRef)
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
		"values":    cmp.Configuration,
		"dryrun":    e.env.DryRun,
	}).Debug("Create component")

	if len(cmp.Secrets) > 0 && !e.env.DryRun {
		err = e.secretsProvider.Write(cmp.Name, cmp.SecretValues)
		if err != nil {
			return err
		}
	}

	_, err = e.env.HelmClient().InstallRelease(
		chartPath,
		e.env.Namespace,
		helm.ValueOverrides([]byte(rawValues)),
		helm.ReleaseName(cmp.Name),
		helm.InstallDryRun(e.env.DryRun),
		helm.InstallReuseName(true),
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
	_, chartPath, err := e.env.ChartLoader.Load(chartRef)
	if err != nil {
		return err
	}

	rawValues, err := cmp.Configuration.YAML()
	if err != nil {
		return err
	}

	if !e.env.DryRun {
		err = e.secretsProvider.Delete(cmp.Name)

		if len(cmp.Secrets) > 0 {
			err = e.secretsProvider.Write(cmp.Name, cmp.SecretValues)
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
		"dryrun":    e.env.DryRun,
	}).Debug("Update component")

	_, err = e.env.HelmClient().UpdateRelease(
		cmp.Name,
		chartPath,
		helm.UpdateValueOverrides([]byte(rawValues)),
		helm.UpgradeDryRun(e.env.DryRun),
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
		"dryrun":  e.env.DryRun,
	}).Debug("Delete component")

	if len(cmp.Secrets) > 0 && !e.env.DryRun {
		err := e.secretsProvider.Delete(cmp.Name)
		if err != nil {
			return err
		}
	}

	if !e.env.DryRun {
		_, err := e.env.HelmClient().DeleteRelease(
			cmp.Name,
			helm.DeletePurge(true),
			helm.DeleteDryRun(e.env.DryRun),
		)
		if err != nil {
			return errors.New(grpc.ErrorDesc(err))
		}
	}

	return nil
}

// diff takes desired and current components, and returns the components to create, update and delete to get from current to desired
func diff(desired, current []*Component) (create, update, delete []*Component) {
	desiredMap := make(map[string]*Component)
	currentMap := make(map[string]*Component)

	for _, c := range desired {
		desiredMap[c.Name] = c
	}
	for _, c := range current {
		currentMap[c.Name] = c
	}

	for name, desiredCmp := range desiredMap {
		if currentCmp, ok := currentMap[name]; ok {
			if !desiredCmp.Equals(currentCmp) {
				update = append(update, desiredCmp)
			}
		} else {
			create = append(create, desiredCmp)
		}
	}

	for name, currentCmp := range currentMap {
		if _, ok := desiredMap[name]; !ok {
			delete = append(delete, currentCmp)
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

// logDifferences logs the Create, Update and Delete w.r.t. current to logf
func logDifferences(current, creates, updates, deletes []*Component, logf func(format string, args ...interface{})) error {
	currentMap := make(map[string]*Component)
	for _, c := range current {
		currentMap[c.Name] = c
	}

	log := func(action string, current, desired *Component) error {
		diff, err := componentDiffText(current, desired)
		if err != nil {
			return err
		}
		logf("%s\n%s", action, diff)
		return nil
	}

	for _, d := range creates {
		if err := log("Create: "+d.Name, nil, d); err != nil {
			return err
		}
	}

	for _, d := range updates {
		c := currentMap[d.Name]
		if err := log("Update: "+d.Name, c, d); err != nil {
			return err
		}
	}

	for _, d := range deletes {
		logf("Delete: %s", d.Name)
	}

	return nil
}
