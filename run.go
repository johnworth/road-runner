package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/cyverse-de/dockerops"
	"github.com/cyverse-de/logcabin"
	"github.com/cyverse-de/messaging"
	"github.com/cyverse-de/model"
)

// JobRunner provides the functionality needed to run jobs.
type JobRunner struct {
	client *messaging.Client
	dckr   *dockerops.Docker
	exit   chan messaging.StatusCode
	job    *model.Job
	status messaging.StatusCode
}

func pullDataImages(dckr *dockerops.Docker, client *messaging.Client, job *model.Job) (messaging.StatusCode, error) {
	var err error
	for _, dc := range job.DataContainers() {
		running(client, job, fmt.Sprintf("Pulling container image %s:%s", dc.Name, dc.Tag))
		if strings.TrimSpace(dc.Auth) == "" {
			err = dckr.Pull(dc.Name, dc.Tag)
		} else {
			running(client, job, fmt.Sprintf("Using auth for pull of %s:%s", dc.Name, dc.Tag))
			err = dckr.PullAuthenticated(dc.Name, dc.Tag, dc.Auth)
		}
		if err != nil {
			running(client, job, fmt.Sprintf("Error pulling container image '%s:%s': %s", dc.Name, dc.Tag, err.Error()))
			return messaging.StatusDockerPullFailed, err
		}
		running(client, job, fmt.Sprintf("Done pulling container image %s:%s", dc.Name, dc.Tag))
	}
	return messaging.Success, err
}

func createDataContainers(dckr *dockerops.Docker, client *messaging.Client, job *model.Job) (messaging.StatusCode, error) {
	var err error
	for _, dc := range job.DataContainers() {
		running(client, job, fmt.Sprintf("Creating data container %s-%s", dc.NamePrefix, job.InvocationID))
		_, err = dckr.CreateDataContainer(&dc, job.InvocationID)
		if err != nil {
			running(client, job, fmt.Sprintf("Error creating data container %s-%s", dc.NamePrefix, job.InvocationID))
			return messaging.StatusDockerPullFailed, err
		}
		running(client, job, fmt.Sprintf("Done creating data container %s-%s", dc.NamePrefix, job.InvocationID))
	}
	return messaging.Success, err
}

func pullStepImages(dckr *dockerops.Docker, client *messaging.Client, job *model.Job) (messaging.StatusCode, error) {
	var err error
	for _, ci := range job.ContainerImages() {
		running(client, job, fmt.Sprintf("Pulling tool container %s:%s", ci.Name, ci.Tag))
		if strings.TrimSpace(ci.Auth) == "" {
			err = dckr.Pull(ci.Name, ci.Tag)
		} else {
			running(client, job, fmt.Sprintf("Using auth for pull of %s:%s", ci.Name, ci.Tag))
			err = dckr.PullAuthenticated(ci.Name, ci.Tag, ci.Auth)
		}
		if err != nil {
			running(client, job, fmt.Sprintf("Error pulling tool container '%s:%s': %s", ci.Name, ci.Tag, err.Error()))
			return messaging.StatusDockerPullFailed, err
		}
		running(client, job, fmt.Sprintf("Done pulling tool container %s:%s", ci.Name, ci.Tag))
	}
	return messaging.Success, err
}

func downloadInputs(dckr *dockerops.Docker, client *messaging.Client, job *model.Job) (messaging.StatusCode, error) {
	var err error
	var exitCode int64
	for idx, input := range job.Inputs() {
		running(client, job, fmt.Sprintf("Downloading %s", input.IRODSPath()))
		exitCode, err = dckr.DownloadInputs(job, &input, idx)
		if exitCode != 0 || err != nil {
			if err != nil {
				running(client, job, fmt.Sprintf("Error downloading %s: %s", input.IRODSPath(), err.Error()))
			} else {
				running(client, job, fmt.Sprintf("Error downloading %s: Transfer utility exited with %d", input.IRODSPath(), exitCode))
			}
			return messaging.StatusInputFailed, err
		}
		running(client, job, fmt.Sprintf("Finished downloading %s", input.IRODSPath()))
	}
	return messaging.Success, err
}

func (r *JobRunner) runAllSteps(exit chan messaging.StatusCode) error {
	var err error
	var exitCode int64

	for idx, step := range r.job.Steps {
		running(r.client, r.job,
			fmt.Sprintf(
				"Running tool container %s:%s with arguments: %s",
				step.Component.Container.Image.Name,
				step.Component.Container.Image.Tag,
				strings.Join(step.Arguments(), " "),
			),
		)

		step.Environment["IPLANT_USER"] = job.Submitter
		step.Environment["IPLANT_EXECUTION_ID"] = job.InvocationID
		exitCode, err = dckr.RunStep(&step, r.job.InvocationID, idx)

		if exitCode != 0 || err != nil {
			if err != nil {
				running(r.client, r.job,
					fmt.Sprintf(
						"Error running tool container %s:%s with arguments '%s': %s",
						step.Component.Container.Image.Name,
						step.Component.Container.Image.Tag,
						strings.Join(step.Arguments(), " "),
						err.Error(),
					),
				)
			} else {
				err = fmt.Errorf(
					"Tool container %s:%s with arguments '%s' exit with code: %d",
					step.Component.Container.Image.Name,
					step.Component.Container.Image.Tag,
					strings.Join(step.Arguments(), " "),
					exitCode,
				)
				running(r.client, r.job, err.Error())
			}
			r.status = messaging.StatusStepFailed
			return err
		}
		running(r.client, r.job,
			fmt.Sprintf("Tool container %s:%s with arguments '%s' finished successfully",
				step.Component.Container.Image.Name,
				step.Component.Container.Image.Tag,
				strings.Join(step.Arguments(), " "),
			),
		)
	}
	return err
}

func (r *JobRunner) uploadOutputs() error {
	var (
		err      error
		exitCode int64
	)

	exitCode, err = dckr.UploadOutputs(r.job)
	if exitCode != 0 || err != nil {
		if err != nil {
			running(r.client, r.job, fmt.Sprintf("Error uploading outputs to %s: %s", r.job.OutputDirectory(), err.Error()))
		} else {
			if r.client == nil {
				logcabin.Warning.Println("client is nil")
			}
			if r.job == nil {
				logcabin.Warning.Println("job is nil")
			}
			od := r.job.OutputDirectory()
			running(r.client, r.job, fmt.Sprintf("Transfer utility exited with a code of %d when uploading outputs to %s", exitCode, od))
		}
		r.status = messaging.StatusOutputFailed
	}

	running(r.client, r.job, fmt.Sprintf("Done uploading outputs to %s", r.job.OutputDirectory()))

	return err
}

// Run executes the job, and returns the exit code on the exit channel.
func Run(client *messaging.Client, dckr *dockerops.Docker, exit chan messaging.StatusCode) {
	runner := &JobRunner{
		client: client,
		dckr:   dckr,
		exit:   exit,
		job:    job,
		status: messaging.Success,
	}

	host, err := os.Hostname()
	if err != nil {
		logcabin.Error.Print(err)
		host = "UNKNOWN"
	}

	// let everyone know the job is running
	running(runner.client, runner.job, fmt.Sprintf("Job %s is running on host %s", runner.job.InvocationID, host))

	transferTrigger, err := os.Create("logs/de-transfer-trigger.log")
	if err != nil {
		logcabin.Error.Print(err)
	} else {
		_, err = transferTrigger.WriteString("This is only used to force HTCondor to transfer files.")
		if err != nil {
			logcabin.Error.Print(err)
		}
	}

	if _, err = os.Stat("iplant.cmd"); err != nil {
		if err = os.Rename("iplant.cmd", "logs/iplant.cmd"); err != nil {
			logcabin.Error.Print(err)
		}
	}

	// Pull the data container images
	if runner.status, err = pullDataImages(runner.dckr, runner.client, job); err != nil {
		logcabin.Error.Print(err)
	}

	// Create the data containers
	if runner.status == messaging.Success {
		if runner.status, err = createDataContainers(runner.dckr, runner.client, job); err != nil {
			logcabin.Error.Print(err)
		}
	}

	// Pull the job step containers
	if runner.status == messaging.Success {
		if runner.status, err = pullStepImages(runner.dckr, runner.client, job); err != nil {
			logcabin.Error.Print(err)
		}
	}

	// // Create the working directory volume
	if runner.status == messaging.Success {
		if _, err = runner.dckr.CreateWorkingDirVolume(job.InvocationID); err != nil {
			logcabin.Error.Print(err)
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		logcabin.Error.Print(err)
	} else {
		voldir := path.Join(wd, dockerops.VOLUMEDIR, "logs")
		logcabin.Info.Printf("path to the volume directory: %s\n", voldir)
		err = os.Mkdir(voldir, 0755)
		if err != nil {
			logcabin.Error.Print(err)
		}

		if err = writeJobSummary(voldir, job); err != nil {
			logcabin.Error.Print(err)
		}

		if err = writeJobParameters(voldir, job); err != nil {
			logcabin.Error.Print(err)
		}
	}
	// If pulls didn't succeed then we can't guarantee that we've got the
	// correct versions of the tools. Don't bother pulling in data in that case,
	// things are already screwed up.
	if runner.status == messaging.Success {
		if runner.status, err = downloadInputs(runner.dckr, runner.client, job); err != nil {
			logcabin.Error.Print(err)
		}
	}

	// Only attempt to run the steps if the input downloads succeeded. No reason
	// to run the steps if there's no/corrupted data to operate on.
	if runner.status == messaging.Success {
		if err = runner.runAllSteps(exit); err != nil {
			logcabin.Error.Print(err)
		}
	}

	// Always attempt to transfer outputs. There might be logs that can help
	// debug issues when the job fails.
	running(runner.client, runner.job, fmt.Sprintf("Beginning to upload outputs to %s", runner.job.OutputDirectory()))
	if err = runner.uploadOutputs(); err != nil {
		logcabin.Error.Print(err)
	}

	// Always inform upstream of the job status.
	if runner.status != messaging.Success {
		fail(runner.client, runner.job, fmt.Sprintf("Job exited with a status of %d", runner.status))
	} else {
		success(runner.client, runner.job)
	}

	exit <- runner.status
}
