package main

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/cyverse-de/model"
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
	nukedImages             []string
	failNukeImage           bool
}

func (t *testOperator) VolumeExists(id string) (bool, error) {
	if !t.failVolumeExists {
		return t.volumeExists, nil
	}
	return t.volumeExists, errors.New("error_volume_exists")
}

func (t *testOperator) RemoveVolume(id string) error {
	t.volumesRemoved = append(t.volumesRemoved, id)
	if t.failVolumeRemoved {
		return errors.New("error_volumes_removed")
	}
	return nil
}

func (t *testOperator) NukeContainer(id string) error {
	t.nukedContainers = append(t.nukedContainers, id)
	if t.failNukeContainer {
		return errors.New("error_nuke_container")
	}
	return nil
}

func (t *testOperator) NukeImage(name, tag string) error {
	t.nukedImages = append(t.nukedImages, fmt.Sprintf("%s:%s", name, tag))
	if t.failNukeImage {
		return errors.New("error_nuke_image")
	}
	return nil
}

func (t *testOperator) ContainersWithLabel(key, value string, all bool) ([]string, error) {
	if t.failContainersWithLabel {
		return nil, errors.New("error_containers")
	}
	return t.containersWithLabel, nil
}

func TestRemoveVolume(t *testing.T) {
	var err error
	operators := []*testOperator{
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

func TestRemoveJobContainers(t *testing.T) {
	op1 := &testOperator{
		containersWithLabel: []string{"foo:bar"},
		nukedContainers:     []string{},
	}
	err := RemoveJobContainers(op1, "foo:bar")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(op1.containersWithLabel, op1.nukedContainers) {
		t.Errorf("nuked containers were %#v, should have been %#v", op1.nukedContainers, op1.containersWithLabel)
	}
	op2 := &testOperator{
		containersWithLabel:     []string{"foo:bar"},
		nukedContainers:         []string{},
		failContainersWithLabel: true,
	}
	err = RemoveJobContainers(op2, "foo:bar")
	if err == nil {
		t.Error("err was nil")
	}
	op3 := &testOperator{
		containersWithLabel: []string{"foo:bar"},
		nukedContainers:     []string{},
		failNukeContainer:   true,
	}
	err = RemoveJobContainers(op3, "foo:bar")
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveDataContainers(t *testing.T) {
	op1 := &testOperator{
		containersWithLabel: []string{"foo:bar"},
		nukedContainers:     []string{},
	}
	err := RemoveDataContainers(op1)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(op1.containersWithLabel, op1.nukedContainers) {
		t.Errorf("nuked containers were %#v, should have been %#v", op1.nukedContainers, op1.containersWithLabel)
	}
	op2 := &testOperator{
		containersWithLabel:     []string{"foo:bar"},
		nukedContainers:         []string{},
		failContainersWithLabel: true,
	}
	err = RemoveDataContainers(op2)
	if err == nil {
		t.Error(err)
	}
	op3 := &testOperator{
		containersWithLabel: []string{"foo:bar"},
		nukedContainers:     []string{},
		failNukeContainer:   true,
	}
	err = RemoveDataContainers(op3)
	if err != nil {
		t.Error(err)
	}
}

func TestRemoveStepContainers(t *testing.T) {
	op1 := &testOperator{
		containersWithLabel: []string{"foo:bar"},
		nukedContainers:     []string{},
	}
	err := RemoveStepContainers(op1)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(op1.containersWithLabel, op1.nukedContainers) {
		t.Errorf("nuked containers were %#v, should have been %#v", op1.nukedContainers, op1.containersWithLabel)
	}
	op2 := &testOperator{
		containersWithLabel:     []string{"foo:bar"},
		nukedContainers:         []string{},
		failContainersWithLabel: true,
	}
	err = RemoveStepContainers(op2)
	if err == nil {
		t.Error("err was nil")
	}
	op3 := &testOperator{
		containersWithLabel: []string{"foo:bar"},
		nukedContainers:     []string{},
		failNukeContainer:   true,
	}
	err = RemoveStepContainers(op3)
	if err != nil {
		t.Error("err was nil")
	}
}

func TestRemoveInputContainers(t *testing.T) {
	op1 := &testOperator{
		containersWithLabel: []string{"foo:bar"},
		nukedContainers:     []string{},
	}
	err := RemoveInputContainers(op1)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(op1.containersWithLabel, op1.nukedContainers) {
		t.Errorf("nuked containers were %#v, should have been %#v", op1.nukedContainers, op1.containersWithLabel)
	}
	op2 := &testOperator{
		containersWithLabel:     []string{"foo:bar"},
		nukedContainers:         []string{},
		failContainersWithLabel: true,
	}
	err = RemoveInputContainers(op2)
	if err == nil {
		t.Error("err was nil")
	}
	op3 := &testOperator{
		containersWithLabel: []string{"foo:bar"},
		nukedContainers:     []string{},
		failNukeContainer:   true,
	}
	err = RemoveInputContainers(op3)
	if err != nil {
		t.Error("err was nil")
	}
}

type dcl struct {
	volumesFrom []model.VolumesFrom
}

func (d *dcl) DataContainers() []model.VolumesFrom {
	return d.volumesFrom
}

func TestRemoveDataContainerImages(t *testing.T) {
	d := &dcl{
		volumesFrom: []model.VolumesFrom{
			{
				Name: "name",
				Tag:  "tag",
			},
		},
	}
	op1 := &testOperator{}
	err := RemoveDataContainerImages(op1, d)
	if err != nil {
		t.Error(err)
	}
	op2 := &testOperator{failNukeImage: true}
	err = RemoveDataContainerImages(op2, d)
	if err != nil {
		t.Error(err)
	}
}
