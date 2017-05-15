package main

import (
	"os/exec"

	"github.com/cyverse-de/messaging"
)

func cleanup() {
	var err error
	downCommand := exec.Command("docker-compose", "-f", "docker-compose.yml", "down", "--rmi", "all", "-v")
	downCommand.Stderr = log.Writer()
	downCommand.Stdout = log.Writer()
	if err = downCommand.Run(); err != nil {
		log.Errorf("%+v", err)
	}
}

// Exit handles clean up when road-runner is killed.
func Exit(exit, finalExit chan messaging.StatusCode) {
	exitCode := <-exit
	log.Warnf("Received an exit code of %d, cleaning up", int(exitCode))
	cleanup()
	finalExit <- exitCode
}
