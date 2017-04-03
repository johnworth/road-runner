package main

import (
	"strconv"

	"github.com/cyverse-de/dockerops"
	"github.com/cyverse-de/messaging"
	"github.com/cyverse-de/model"
	"github.com/pkg/errors"
)

// RemoveVolume removes the docker volume with the provided volume identifier.
func RemoveVolume(id string) error {
	var (
		err       error
		hasVolume bool
	)
	hasVolume, err = dckr.VolumeExists(id)
	if err != nil {
		return errors.Wrap(err, "failed to check for volume existence")
	}
	if hasVolume {
		log.Infof("removing volume: %s", id)
		if err = dckr.RemoveVolume(id); err != nil {
			return errors.Wrap(err, "failed to remove volume")
		}
	}
	return nil
}

// RemoveJobContainers removes containers based on their job label.
func RemoveJobContainers(id string) error {
	log.Infof("Finding all containers with the label %s=%s", model.DockerLabelKey, id)
	jobContainers, err := dckr.ContainersWithLabel(model.DockerLabelKey, id, true)
	if err != nil {
		return errors.Wrapf(err, "failed to find containers with label %s=%s", model.DockerLabelKey, id)
	}
	for _, jc := range jobContainers {
		log.Infof("Nuking container %s", jc)
		err = dckr.NukeContainer(jc)
		if err != nil {
			log.Errorf("%+v", errors.Wrapf(err, "failed to remove container %s", jc))
		}
	}
	return nil
}

// RemoveDataContainers attempts to remove all data containers.
func RemoveDataContainers() error {
	log.Infoln("Finding all data containers")
	dataContainers, err := dckr.ContainersWithLabel(dockerops.TypeLabel, strconv.Itoa(dockerops.DataContainer), true)
	if err != nil {
		return errors.Wrapf(err, "failed to find containers with label %s=%s", dockerops.TypeLabel, strconv.Itoa(dockerops.DataContainer))
	}
	for _, dc := range dataContainers {
		log.Infof("Nuking data container %s", dc)
		err = dckr.NukeContainer(dc)
		if err != nil {
			log.Errorf("%+v", errors.Wrapf(err, "failed to remove container %s", dc))
		}
	}
	return nil
}

// RemoveStepContainers attempts to remove all step containers.
func RemoveStepContainers() error {
	log.Infoln("Finding all step containers")
	stepContainers, err := dckr.ContainersWithLabel(dockerops.TypeLabel, strconv.Itoa(dockerops.StepContainer), true)
	if err != nil {
		return errors.Wrapf(err, "failed to find containers with label %s=%s", dockerops.TypeLabel, strconv.Itoa(dockerops.StepContainer))
	}
	for _, sc := range stepContainers {
		log.Infof("Nuking step container %s", sc)
		err = dckr.NukeContainer(sc)
		if err != nil {
			log.Errorf("%+v", errors.Wrapf(err, "failed to remove container %s", sc))
		}
	}
	return nil
}

// RemoveInputContainers attempts to remove all input containers.
func RemoveInputContainers() error {
	log.Infoln("Finding all input containers")
	inputContainers, err := dckr.ContainersWithLabel(dockerops.TypeLabel, strconv.Itoa(dockerops.InputContainer), true)
	if err != nil {
		return errors.Wrapf(err, "failed to find containers with label %s=%s", dockerops.TypeLabel, strconv.Itoa(dockerops.InputContainer))
	}
	for _, ic := range inputContainers {
		log.Infof("Nuking input container %s", ic)
		err = dckr.NukeContainer(ic)
		if err != nil {
			log.Errorf("%+v", errors.Wrapf(err, "failed to remove container %s", ic))
		}
	}
	return nil
}

// RemoveDataContainerImages removes the images for the data containers.
func RemoveDataContainerImages() error {
	var err error
	for _, dc := range job.DataContainers() {
		log.Infof("Nuking image %s:%s", dc.Name, dc.Tag)
		err = dckr.NukeImage(dc.Name, dc.Tag)
		if err != nil {
			log.Errorf("%+v", errors.Wrapf(err, "failed to remove image %s:%s", dc.Name, dc.Tag))
		}
	}
	return nil
}

// cleanup encapsulates common job clean up tasks.
func cleanup(job *model.Job) {
	var err error
	log.Infof("Performing aggressive clean up routine...")
	if err = RemoveInputContainers(); err != nil {
		log.Errorf("%+v", err)
	}
	if err = RemoveStepContainers(); err != nil {
		log.Errorf("%+v", err)
	}
	if err = RemoveDataContainers(); err != nil {
		log.Errorf("%+v", err)
	}
	if err = RemoveVolume(job.InvocationID); err != nil {
		log.Errorf("%+v", err)
	}
}

// Exit handles clean up when road-runner is killed.
func Exit(exit, finalExit chan messaging.StatusCode) {
	var err error
	exitCode := <-exit
	log.Warnf("Received an exit code of %d, cleaning up", int(exitCode))
	switch exitCode {
	case messaging.StatusKilled:
		//Annihilate the input/steps/data containers even if they're running,
		//but allow the output containers to run. Yanking the rug out from the
		//containers should force the Run() function to 'fall through' to any clean
		//up steps.
		if err = RemoveDataContainerImages(); err != nil {
			log.Errorf("%+v", err)
		}
		if err = RemoveInputContainers(); err != nil {
			log.Errorf("%+v", err)
		}
		if err = RemoveStepContainers(); err != nil {
			log.Errorf("%+v", err)
		}
		if err = RemoveDataContainers(); err != nil {
			log.Errorf("%+v", err)
		}
		if err = RemoveVolume(job.InvocationID); err != nil {
			log.Errorf("%+v", err)
		}
		if err = RemoveJobContainers(job.InvocationID); err != nil {
			log.Errorf("%+v", err)
		}
	default:
		if err = RemoveJobContainers(job.InvocationID); err != nil {
			log.Errorf("%+v", err)
		}
		if err = RemoveVolume(job.InvocationID); err != nil {
			log.Errorf("%+v", err)
		}
	}
	finalExit <- exitCode
}
