package landscaper

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
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
	env *Environment
}

// NewExecutor is a factory method to create a new Executor
func NewExecutor(env *Environment) (Executor, error) {
	if err := env.EnsureHelmClient(); err != nil {
		return nil, err
	}

	return &executor{env: env}, nil
}

// Apply transforms the current state into the desired state
func (e *executor) Apply(desired, current []*Component) error {
	create, update, delete := diff(desired, current)

	logrus.WithFields(logrus.Fields{
		"desired": desired,
		"current": current,
		"create":  create,
		"update":  update,
		"delete":  delete,
		"dryrun":  e.env.DryRun,
	}).Info("apply desired state")

	for _, cmp := range delete {
		if err := e.DeleteComponent(cmp); err != nil {
			return err
		}
	}

	for _, cmp := range create {
		if err := e.CreateComponent(cmp); err != nil {
			return err
		}
	}

	for _, cmp := range update {
		if err := e.UpdateComponent(cmp); err != nil {
			return err
		}
	}

	return nil
}

// CreateComponent creates the given Component
func (e *executor) CreateComponent(cmp *Component) error {
	chartRef := fmt.Sprintf("%s/%s", e.env.HelmRepositoryName, cmp.Release.Chart)

	// We need to ensure the chart is available on the local system. LoadChart will ensure
	// this is the case by downloading the chart if it is not there yet
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
		"chartRef":  chartRef,
		"chartPath": chartPath,
		"values":    cmp.Configuration,
		"dryrun":    e.env.DryRun,
	}).Info("create component")

	_, err = e.env.HelmClient.InstallRelease(
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
	chartRef := fmt.Sprintf("%s/%s", e.env.HelmRepositoryName, cmp.Release.Chart)

	// We need to ensure the chart is available on the local system. LoadChart will ensure
	// this is the case by downloading the chart if it is not there yet
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
		"chartRef":  chartRef,
		"chartPath": chartPath,
		"values":    cmp.Configuration,
		"dryrun":    e.env.DryRun,
	}).Info("update component")

	_, err = e.env.HelmClient.UpdateRelease(
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
	}).Info("delete component")

	// TODO: work around https://github.com/kubernetes/helm/pull/1527 as long as needed
	if e.env.DryRun {
		return nil
	}

	_, err := e.env.HelmClient.DeleteRelease(
		cmp.Name,
		helm.DeletePurge(true),
		helm.DeleteDryRun(e.env.DryRun),
	)
	if err != nil {
		return errors.New(grpc.ErrorDesc(err))
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
