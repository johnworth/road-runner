package main

import (
	"errors"
	"fmt"
	"testing"
)

type testOperator struct {
	volumeExists            bool
	failVolumeExists        bool
	failRemoveVolume        bool
	volumesRemoved          []string
	failVolumeRemoved       bool
	failContainersWithLabel bool
	containersWithLabel     []string
	failNukeContainer       bool
	nukedContainers         []string
}

func (t testOperator) VolumeExists(id string) (bool, error) {
	if !t.failVolumeExists {
		return t.volumeExists, nil
	}
	return t.volumeExists, errors.New("error_volume_exists")
}

func (t testOperator) RemoveVolume(id string) error {
	t.volumesRemoved = append(t.volumesRemoved, id)
	if t.failVolumeRemoved {
		return errors.New("error_volumes_removed")
	}
	return nil
}

func (t testOperator) NukeContainer(id string) error {
	fmt.Println(id)
	t.nukedContainers = append(t.nukedContainers, id)
	fmt.Println(t.nukedContainers)
	if t.failNukeContainer {
		return errors.New("error_nuke_container")
	}
	return nil
}

func (t testOperator) NukeImage(name, tag string) error {
	return nil
}

func (t testOperator) ContainersWithLabel(key, value string, all bool) ([]string, error) {
	if t.failContainersWithLabel {
		return nil, errors.New("error_containers")
	}
	return t.containersWithLabel, nil
}

func TestRemoveVolume(t *testing.T) {
	var err error
	operators := []testOperator{
		{
			volumeExists:      false,
			failVolumeExists:  false,
			volumesRemoved:    make([]string, 0),
			failVolumeRemoved: false,
		},
		{
			volumeExists:      true,
			failVolumeExists:  false,
			volumesRemoved:    make([]string, 0),
			failVolumeRemoved: false,
		},
		{
			volumeExists:      true,
			failVolumeExists:  true,
			volumesRemoved:    make([]string, 0),
			failVolumeRemoved: false,
		},
		{
			volumeExists:      true,
			failVolumeExists:  true,
			volumesRemoved:    make([]string, 0),
			failVolumeRemoved: true,
		},
		{
			volumeExists:      false,
			failVolumeExists:  true,
			volumesRemoved:    make([]string, 0),
			failVolumeRemoved: true,
		},
		{
			volumeExists:      false,
			failVolumeExists:  false,
			volumesRemoved:    make([]string, 0),
			failVolumeRemoved: true,
		},
		{
			volumeExists:      true,
			failVolumeExists:  false,
			volumesRemoved:    make([]string, 0),
			failVolumeRemoved: true,
		},
		{
			volumeExists:      false,
			failVolumeExists:  true,
			volumesRemoved:    make([]string, 0),
			failVolumeRemoved: false,
		},
	}
	for _, op := range operators {
		err = RemoveVolume(op, "test_id")
		if !op.volumeExists && !op.failVolumeExists && !op.failVolumeRemoved && err != nil {
			t.Error("err was not nil")
		}
		if op.volumeExists && !op.failVolumeExists && !op.failVolumeRemoved && err != nil {
			t.Error("err was not nil")
		}
		if op.volumeExists && op.failVolumeExists && !op.failVolumeRemoved && err == nil {
			t.Error("err was nil")
		}
		if op.volumeExists && op.failVolumeExists && op.failVolumeRemoved && err == nil {
			t.Error("err was nil")
		}
		if !op.volumeExists && op.failVolumeExists && op.failVolumeRemoved && err == nil {
			t.Error("err was nil")
		}
		// Can't fail to remove a volume that doesn't exist.
		if !op.volumeExists && !op.failVolumeExists && op.failVolumeRemoved && err != nil {
			t.Error("err was not nil")
		}
		if op.volumeExists && !op.failVolumeExists && op.failVolumeRemoved && err == nil {
			t.Error("err was nil")
		}
		if !op.volumeExists && op.failVolumeExists && !op.failVolumeRemoved && err == nil {
			t.Error("err was nil")
		}
	}
}
