package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/cyverse-de/configurate"
	"github.com/cyverse-de/messaging"
	"github.com/cyverse-de/model"

	"github.com/spf13/viper"
)

var (
	s   *model.Job
	cfg *viper.Viper
)

func shouldrun() bool {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "" {
		return true
	}
	return false
}

func uri() string {
	return "http://dind:2375"
}

func JSONData() ([]byte, error) {
	f, err := os.Open("test/test_runner.json")
	if err != nil {
		return nil, err
	}
	c, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return c, err
}

func _inittests(t *testing.T, memoize bool) *model.Job {
	var err error
	if s == nil || !memoize {
		cfg, err = configurate.Init("test/test_config.yaml")
		if err != nil {
			t.Fatal(err)
		}
		cfg.Set("irods.base", "/path/to/irodsbase")
		cfg.Set("irods.host", "hostname")
		cfg.Set("irods.port", "1247")
		cfg.Set("irods.user", "user")
		cfg.Set("irods.pass", "pass")
		cfg.Set("irods.zone", "test")
		cfg.Set("irods.resc", "")
		cfg.Set("condor.log_path", "/path/to/logs")
		cfg.Set("condor.porklock_tag", "test")
		cfg.Set("condor.filter_files", "foo,bar,baz,blippy")
		cfg.Set("condor.request_disk", "0")
		data, err := JSONData()
		if err != nil {
			t.Error(err)
		}
		s, err = model.NewFromData(cfg, data)
		if err != nil {
			t.Error(err)
		}
	}
	return s
}

func inittests(t *testing.T) *model.Job {
	return _inittests(t, true)
}

func GetClient(t *testing.T) *messaging.Client {
	var err error
	if client != nil {
		return client
	}
	client, err = messaging.NewClient(messagingURI(), false)
	if err != nil {
		t.Error(err)
	}
	client.SetupPublishing(messagingExchangeName())
	go client.Listen()
	return client
}

func messagingURI() string {
	ret := cfg.GetString("amqp.uri")
	return ret
}

func messagingExchangeName() string {
	ret := cfg.GetString("amqp.exchange.name")
	return ret
}

func messagingExchangeType() string {
	ret := cfg.GetString("amqp.exchange.type")
	return ret
}

func TestRegisterStopRequestListener(t *testing.T) {
	if !shouldrun() {
		return
	}
	client := GetClient(t)
	invID := "test"
	exit := make(chan messaging.StatusCode)
	RegisterStopRequestListener(client, exit, invID)
	err := client.SendStopRequest(invID, "test", "this is a test")
	if err != nil {
		t.Error(err)
	}
	actual := <-exit
	if actual != messaging.StatusKilled {
		t.Errorf("StatusCode was %d instead of %d", int64(actual), int64(messaging.StatusKilled))
	}
}

func TestCopyJobFile(t *testing.T) {
	uuid := "00000000-0000-0000-0000-000000000000"
	from := path.Join("test", fmt.Sprintf("%s.json", uuid))
	to := "/tmp"
	err := copyJobFile(uuid, from, to)
	if err != nil {
		t.Error(err)
	}
	tmpPath := path.Join(to, fmt.Sprintf("%s.json", uuid))
	if _, err := os.Open(tmpPath); err != nil {
		t.Error(err)
	} else {
		if err = os.Remove(tmpPath); err != nil {
			t.Error(err)
		}
	}
}

func TestDeleteJobFile(t *testing.T) {
	uuid := "00000000-0000-0000-0000-000000000000"
	from := path.Join("test", fmt.Sprintf("%s.json", uuid))
	to := "/tmp"
	err := copyJobFile(uuid, from, to)
	if err != nil {
		t.Error(err)
	}
	deleteJobFile(uuid, to)
	tmpPath := path.Join(to, fmt.Sprintf("%s.json", uuid))
	if _, err := os.Open(tmpPath); err == nil {
		t.Errorf("tmpPath %s existed after deleteJobFile() was called", tmpPath)
	}
}

func TestJobWithoutCancellationWarning(t *testing.T) {
	if determineCancellationWarningBuffer(59*time.Second) != 0 {
		t.Error("A timeout warning message would be produced when it shouldn't")
	}
}

func TestJobWithMinimumWarningBuffer(t *testing.T) {
	cancellationWarningBuffer := determineCancellationWarningBuffer(61 * time.Second)
	if cancellationWarningBuffer == 0 {
		t.Error("A timeout warning would" +
			" not be produced when it should")
	} else if cancellationWarningBuffer != minCancellationBuffer {
		t.Errorf(
			"Unexpected duration between cancellation warning and job cancellation: %s",
			cancellationWarningBuffer.String(),
		)
	}
}

func TestJobWithDefaultWarningBuffer(t *testing.T) {
	cancellationWarningBuffer := determineCancellationWarningBuffer(500 * time.Second)
	if cancellationWarningBuffer == 0 {
		t.Error("A timeout warning would not be produced when it should")
	} else if cancellationWarningBuffer != 100*time.Second {
		t.Errorf(
			"Unexpected duration between cancellation warning and job cancellation: %s",
			cancellationWarningBuffer.String(),
		)
	}
}

func TestJobWithMaximumWarningBuffer(t *testing.T) {
	cancellationWarningBuffer := determineCancellationWarningBuffer(30 * time.Minute)
	if cancellationWarningBuffer == 0 {
		t.Error("A timeout warning would not be produced when it should")
	} else if cancellationWarningBuffer != maxCancellationBuffer {
		t.Errorf(
			"Unexpected duration between cancellation warning and job cancellation: %s",
			cancellationWarningBuffer.String(),
		)
	}
}
