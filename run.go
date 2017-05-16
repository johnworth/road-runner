package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/cyverse-de/messaging"
	"github.com/cyverse-de/model"
	"github.com/cyverse-de/road-runner/dcompose"
	"github.com/cyverse-de/road-runner/fs"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// JobRunner provides the functionality needed to run jobs.
type JobRunner struct {
	client JobUpdatePublisher
	exit   chan messaging.StatusCode
	job    *model.Job
	status messaging.StatusCode
}

// JobUpdatePublisher is the interface for types that need to publish a job
// update.
type JobUpdatePublisher interface {
	PublishJobUpdate(m *messaging.UpdateMessage) error
}

func createDataContainers(client JobUpdatePublisher, job *model.Job, cfg *viper.Viper) (messaging.StatusCode, error) {
	var (
		err error
	)
	composePath := cfg.GetString("docker-compose.path")
	for index := range job.DataContainers() {
		running(client, job, fmt.Sprintf("creating data container data_%d", index))
		dataCommand := exec.Command(composePath, "-f", "docker-compose.yml", "up", fmt.Sprintf("data_%d", index))
		dataCommand.Env = os.Environ()
		dataCommand.Stderr = log.Writer()
		dataCommand.Stdout = log.Writer()
		if err = dataCommand.Run(); err != nil {
			running(client, job, fmt.Sprintf("error creating data container data_%d: %s", index, err.Error()))
			return messaging.StatusDockerCreateFailed, errors.Wrapf(err, "failed to create data container data_%d", index)
		}
		running(client, job, fmt.Sprintf("finished creating data container data_%d", index))
	}
	return messaging.Success, nil
}

func downloadInputs(client JobUpdatePublisher, job *model.Job, cfg *viper.Viper) (messaging.StatusCode, error) {
	var (
		err      error
		exitCode int64
	)
	env := os.Environ()
	env = append(env, fmt.Sprintf("VAULT_ADDR=%s", cfg.GetString("vault.url")))
	env = append(env, fmt.Sprintf("VAULT_TOKEN=%s", cfg.GetString("vault.token")))
	composePath := cfg.GetString("docker-compose.path")
	for index, input := range job.Inputs() {
		running(client, job, fmt.Sprintf("Downloading %s", input.IRODSPath()))
		downloadCommand := exec.Command(composePath, "-f", "docker-compose.yml", "up", fmt.Sprintf("input_%d", index))
		downloadCommand.Env = env
		downloadCommand.Stderr = log.Writer()
		downloadCommand.Stdout = log.Writer()
		if err = downloadCommand.Run(); err != nil {
			running(client, job, fmt.Sprintf("error downloading %s: %s", input.IRODSPath(), err.Error()))
			return messaging.StatusInputFailed, errors.Wrapf(err, "failed to download %s with an exit code of %d", input.IRODSPath(), exitCode)
		}
		running(client, job, fmt.Sprintf("finished downloading %s", input.IRODSPath()))
	}
	return messaging.Success, nil
}

func runAllSteps(client JobUpdatePublisher, job *model.Job, cfg *viper.Viper, exit chan messaging.StatusCode) (messaging.StatusCode, error) {
	var err error

	for idx, step := range job.Steps {
		running(client, job,
			fmt.Sprintf(
				"Running tool container %s:%s with arguments: %s",
				step.Component.Container.Image.Name,
				step.Component.Container.Image.Tag,
				strings.Join(step.Arguments(), " "),
			),
		)
		composePath := cfg.GetString("docker-compose.path")
		runCommand := exec.Command(composePath, "-f", "docker-compose.yml", "up", fmt.Sprintf("step_%d", idx))
		runCommand.Env = os.Environ()
		runCommand.Stdout = log.Writer()
		runCommand.Stderr = log.Writer()
		err = runCommand.Run()

		if err != nil {
			running(client, job,
				fmt.Sprintf(
					"Error running tool container %s:%s with arguments '%s': %s",
					step.Component.Container.Image.Name,
					step.Component.Container.Image.Tag,
					strings.Join(step.Arguments(), " "),
					err.Error(),
				),
			)
			return messaging.StatusStepFailed, err
		}

		running(client, job,
			fmt.Sprintf("Tool container %s:%s with arguments '%s' finished successfully",
				step.Component.Container.Image.Name,
				step.Component.Container.Image.Tag,
				strings.Join(step.Arguments(), " "),
			),
		)
	}
	return messaging.Success, err
}

func uploadOutputs(client JobUpdatePublisher, job *model.Job, cfg *viper.Viper) (messaging.StatusCode, error) {
	var err error
	composePath := cfg.GetString("docker-compose.path")
	outputCommand := exec.Command(composePath, "-f", "docker-compose.yml", "up", "upload_outputs")
	outputCommand.Env = os.Environ()
	outputCommand.Env = []string{
		fmt.Sprintf("VAULT_ADDR=%s", cfg.GetString("vault.url")),
		fmt.Sprintf("VAULT_TOKEN=%s", cfg.GetString("vault.token")),
	}
	outputCommand.Stdout = log.Writer()
	outputCommand.Stderr = log.Writer()
	err = outputCommand.Run()

	if err != nil {
		running(client, job, fmt.Sprintf("Error uploading outputs to %s: %s", job.OutputDirectory(), err.Error()))
		return messaging.StatusOutputFailed, errors.Wrapf(err, "failed to upload outputs to %s", job.OutputDirectory())
	}

	running(client, job, fmt.Sprintf("Done uploading outputs to %s", job.OutputDirectory()))
	return messaging.Success, nil
}

// Run executes the job, and returns the exit code on the exit channel.
func Run(client JobUpdatePublisher, job *model.Job, cfg *viper.Viper, exit chan messaging.StatusCode) {
	runner := &JobRunner{
		client: client,
		exit:   exit,
		job:    job,
		status: messaging.Success,
	}

	host, err := os.Hostname()
	if err != nil {
		log.Error(err)
		host = "UNKNOWN"
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Error(err)
	}

	voldir := path.Join(cwd, dcompose.VOLUMEDIR, "logs")
	log.Infof("path to the volume directory: %s\n", voldir)
	err = os.MkdirAll(voldir, 0755)
	if err != nil {
		log.Error(err)
	}

	// let everyone know the job is running
	running(runner.client, runner.job, fmt.Sprintf("Job %s is running on host %s", runner.job.InvocationID, host))

	transferTrigger, err := os.Create(path.Join(cwd, dcompose.VOLUMEDIR, "logs", "de-transfer-trigger.log"))
	if err != nil {
		log.Error(err)
	} else {
		_, err = transferTrigger.WriteString("This is only used to force HTCondor to transfer files.")
		if err != nil {
			log.Error(err)
		}
	}

	if _, err = os.Stat("iplant.cmd"); err != nil {
		if err = os.Rename("iplant.cmd", path.Join(cwd, dcompose.VOLUMEDIR, "logs", "iplant.cmd")); err != nil {
			log.Error(err)
		}
	}

	pullCommand := exec.Command("docker-compose", "-f", "docker-compose.yml", "pull", "--parallel")
	pullCommand.Env = os.Environ()
	pullCommand.Dir = cwd
	pullCommand.Stdout = log.Writer()
	pullCommand.Stderr = log.Writer()
	err = pullCommand.Run()
	if err != nil {
		log.Error(err)
		runner.status = messaging.StatusDockerPullFailed
	}

	if err = fs.WriteJobSummary(fs.FS, voldir, job); err != nil {
		log.Error(err)
	}

	if err = fs.WriteJobParameters(fs.FS, voldir, job); err != nil {
		log.Error(err)
	}

	if runner.status == messaging.Success {
		if runner.status, err = createDataContainers(runner.client, job, cfg); err != nil {
			log.Error(err)
		}
	}

	// If pulls didn't succeed then we can't guarantee that we've got the
	// correct versions of the tools. Don't bother pulling in data in that case,
	// things are already screwed up.
	if runner.status == messaging.Success {
		if runner.status, err = downloadInputs(runner.client, job, cfg); err != nil {
			log.Error(err)
		}
	}
	// Only attempt to run the steps if the input downloads succeeded. No reason
	// to run the steps if there's no/corrupted data to operate on.
	if runner.status == messaging.Success {
		if runner.status, err = runAllSteps(runner.client, job, cfg, exit); err != nil {
			log.Error(err)
		}
	}
	// Always attempt to transfer outputs. There might be logs that can help
	// debug issues when the job fails.
	running(runner.client, runner.job, fmt.Sprintf("Beginning to upload outputs to %s", runner.job.OutputDirectory()))
	if runner.status, err = uploadOutputs(runner.client, job, cfg); err != nil {
		log.Error(err)
	}
	// Always inform upstream of the job status.
	if runner.status != messaging.Success {
		fail(runner.client, runner.job, fmt.Sprintf("Job exited with a status of %d", runner.status))
	} else {
		success(runner.client, runner.job)
	}
	exit <- runner.status
}
