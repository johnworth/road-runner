package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cyverse-de/messaging"
	"github.com/cyverse-de/model"
	"github.com/docker/docker/api/types"
)

var testJob = &model.Job{
	ID:           "test-job-id",
	InvocationID: "test-invocation-id",
	Steps: []model.Step{
		{
			Type:       "condor",
			StdinPath:  "/stdin/path",
			StdoutPath: "/stdout/path",
			StderrPath: "/stderr/path",
			LogFile:    "/logfile/path",
			Environment: map[string]string{
				"FOO": "BAR",
				"BAZ": "1",
			},
			Input: []model.StepInput{
				{
					ID:           "step-input-1",
					Multiplicity: "wut",
					Name:         "step-input-name-1",
					Property:     "step-input-property-1",
					Retain:       false,
					Type:         "step-input-type-1",
					Value:        "step-input-value-1",
				},
				{
					ID:           "step-input-2",
					Multiplicity: "wut2",
					Name:         "step-input-name-2",
					Property:     "step-input-property-2",
					Retain:       false,
					Type:         "step-input-type-2",
					Value:        "step-input-value-2",
				},
			},
			Config: model.StepConfig{
				Params: []model.StepParam{
					{
						ID:    "step-param-1",
						Name:  "step-param-name-1",
						Value: "step-param-value-1",
						Order: 0,
					},
					{
						ID:    "step-param-2",
						Name:  "step-param-name-2",
						Value: "step-param-value-2",
						Order: 1,
					},
				},
			},
			Component: model.StepComponent{
				Container: model.Container{
					ID:   "container-id-1",
					Name: "container-name-1",
					Image: model.ContainerImage{
						ID:   "container-image-1",
						Name: "container-image-name-1",
						Tag:  "container-image-tag-1",
					},
					VolumesFrom: []model.VolumesFrom{
						{
							Tag:           "tag1",
							Name:          "name1",
							HostPath:      "/host/path1",
							ContainerPath: "/container/path1",
						},
						{
							Tag:           "tag2",
							Name:          "name2",
							HostPath:      "/host/path2",
							ContainerPath: "/container/path2",
						},
					},
				},
			},
		},
	},
}

type DockerOpsTester struct {
	failCreateDataContainer    bool
	failCreateWorkingDirVolume bool
	failDownloadInputs         bool
	failPull                   bool
	failPullAuthenticated      bool
	failRunStep                bool
	failUploadOutputs          bool
	createdDataContainers      []*model.VolumesFrom
	createdWorkingDirVolumes   []string
	downloadedInputs           []*model.StepInput
	pulledImages               []string
	pulledAuthenticatedImages  []string
	stepsRun                   []*model.Step
	outputsUploaded            []string
}

func NewDockerOpsTester() *DockerOpsTester {
	return &DockerOpsTester{
		createdDataContainers:     []*model.VolumesFrom{},
		createdWorkingDirVolumes:  []string{},
		downloadedInputs:          []*model.StepInput{},
		pulledImages:              []string{},
		pulledAuthenticatedImages: []string{},
		stepsRun:                  []*model.Step{},
		outputsUploaded:           []string{},
	}
}

func (d *DockerOpsTester) CreateDataContainer(vf *model.VolumesFrom, invocationID string) (string, error) {
	if d.failCreateDataContainer {
		return "", errors.New("failed to create data container")
	}
	d.createdDataContainers = append(d.createdDataContainers, vf)
	return "test", nil
}

func (d *DockerOpsTester) CreateWorkingDirVolume(volumeID string) (types.Volume, error) {
	if d.failCreateWorkingDirVolume {
		return types.Volume{}, errors.New("failed to create working directory volume")
	}
	d.createdWorkingDirVolumes = append(d.createdWorkingDirVolumes, volumeID)
	return types.Volume{}, nil
}

func (d *DockerOpsTester) DownloadInputs(j *model.Job, i *model.StepInput, index int) (int64, error) {
	if d.failDownloadInputs {
		return 99, errors.New("failed to download input files")
	}
	d.downloadedInputs = append(d.downloadedInputs, i)
	return 0, nil
}

func (d *DockerOpsTester) Pull(name, tag string) error {
	if d.failPull {
		return errors.New("failed to pull image")
	}
	d.pulledImages = append(d.pulledImages, fmt.Sprintf("%s:%s", name, tag))
	return nil
}

func (d *DockerOpsTester) PullAuthenticated(name, tag, auth string) error {
	if d.failPullAuthenticated {
		return errors.New("failed to pull image while authenticated")
	}
	d.pulledAuthenticatedImages = append(d.pulledAuthenticatedImages, fmt.Sprintf("%s:%s", name, tag))
	return nil
}

func (d *DockerOpsTester) RunStep(s *model.Step, invocationID string, stepIndex int) (int64, error) {
	if d.failRunStep {
		return 99, errors.New("failed to run step")
	}
	d.stepsRun = append(d.stepsRun, s)
	return 0, nil
}

func (d *DockerOpsTester) UploadOutputs(j *model.Job) (int64, error) {
	if d.failUploadOutputs {
		return 99, errors.New("failed to upload outputs")
	}
	d.outputsUploaded = append(d.outputsUploaded, j.InvocationID)
	return 0, nil
}

func TestPullDataImages(t *testing.T) {
	d := NewDockerOpsTester()
	u := NewTestJobUpdatePublisher(false)
	sc, err := pullDataImages(d, u, testJob)
	if err != nil {
		t.Error(err)
	}
	if sc != messaging.Success {
		t.Errorf("status code was %d instead of %d", sc, messaging.Success)
	}
}

func TestCreateDataContainers(t *testing.T) {
	d := NewDockerOpsTester()
	u := NewTestJobUpdatePublisher(false)
	sc, err := createDataContainers(d, u, testJob)
	if err != nil {
		t.Error(err)
	}
	if sc != messaging.Success {
		t.Errorf("status code was %d instead of %d", sc, messaging.Success)
	}
}

func TestPullStepImages(t *testing.T) {
	d := NewDockerOpsTester()
	u := NewTestJobUpdatePublisher(false)
	sc, err := pullStepImages(d, u, testJob)
	if err != nil {
		t.Error(err)
	}
	if sc != messaging.Success {
		t.Errorf("status code was %d instead of %d", sc, messaging.Success)
	}
}

func TestDownloadInputs(t *testing.T) {
	d := NewDockerOpsTester()
	u := NewTestJobUpdatePublisher(false)
	sc, err := downloadInputs(d, u, testJob)
	if err != nil {
		t.Error(err)
	}
	if sc != messaging.Success {
		t.Errorf("status code was %d instead of %d", sc, messaging.Success)
	}
}

func TestRunAllSteps(t *testing.T) {
	d := NewDockerOpsTester()
	u := NewTestJobUpdatePublisher(false)
	e := make(chan messaging.StatusCode, 0)
	sc, err := runAllSteps(d, u, testJob, e)
	if err != nil {
		t.Error(err)
	}
	if sc != messaging.Success {
		t.Errorf("status code was %d instead of %d", sc, messaging.Success)
	}
}

func TestUploadOutputs(t *testing.T) {
	d := NewDockerOpsTester()
	u := NewTestJobUpdatePublisher(false)
	sc, err := uploadOutputs(d, u, testJob)
	if err != nil {
		t.Error(err)
	}
	if sc != messaging.Success {
		t.Errorf("status code was %d instead of %d", sc, messaging.Success)
	}
}
