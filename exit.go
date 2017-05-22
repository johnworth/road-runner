package main

import (
	"os/exec"

	"github.com/cyverse-de/messaging"
	"github.com/spf13/viper"
)

// This is called from main() as well, which is why it's a separate function.
func cleanup(cfg *viper.Viper) {
	var err error
	downCommand := exec.Command(
		cfg.GetString("docker-compose.path"),
		"-f", "docker-compose.yml",
		"down",         // down seems to be the only way to clean up images with d-c.
		"--rmi", "all", // tells d-c to clean up all images used by a service
		"-v", // not verbose, tells docker-compose to clean up related volumes.
	)
	downCommand.Stderr = log.Writer()
	downCommand.Stdout = log.Writer()
	if err = downCommand.Run(); err != nil {
		log.Errorf("%+v", err)
	}
}

// Exit handles clean up when road-runner is killed.
func Exit(cfg *viper.Viper, exit, finalExit chan messaging.StatusCode) {
	exitCode := <-exit
	log.Warnf("Received an exit code of %d, cleaning up", int(exitCode))
	cleanup(cfg)
	finalExit <- exitCode
}
