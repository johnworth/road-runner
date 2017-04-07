package main

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/cyverse-de/model"
)

func TestWriteCSV(t *testing.T) {
	expected := `test0,test0,test0
test1,test1,test1
test2,test2,test2
`
	records := [][]string{
		{"test0", "test0", "test0"},
		{"test1", "test1", "test1"},
		{"test2", "test2", "test2"},
	}
	outputPath := path.Join("test", "TestWriteCSV.csv")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		t.Error(err)
	}
	if err = writeCSV(outputFile, records); err != nil {
		t.Error(err)
	}
	inBytes, err := ioutil.ReadFile(outputPath)
	if err != nil {
		t.Error(err)
	}
	actual := string(inBytes)
	if actual != expected {
		t.Errorf("Contents of %s were:\n%s\n\tinstead of:\n%s\n", outputPath, actual, expected)
	}
	if err = os.Remove(outputPath); err != nil {
		t.Error(err)
	}
}

func TestWriteJobSummary(t *testing.T) {
	j := &model.Job{
		InvocationID: "07b04ce2-7757-4b21-9e15-0b4c2f44be26",
		Name:         "Echo_test",
		AppID:        "c7f05682-23c8-4182-b9a2-e09650a5f49b",
		AppName:      "Word Count",
		Submitter:    "test_this_is_a_test",
	}
	expected := `Job ID,07b04ce2-7757-4b21-9e15-0b4c2f44be26
Job Name,Echo_test
Application ID,c7f05682-23c8-4182-b9a2-e09650a5f49b
Application Name,Word Count
Submitted By,test_this_is_a_test
`
	if err := writeJobSummary("test", j); err != nil {
		t.Error(err)
	}
	outPath := "test/JobSummary.csv"
	input, err := ioutil.ReadFile(outPath)
	if err != nil {
		t.Error(err)
	}
	actual := string(input)
	if actual != expected {
		t.Errorf("Contents of %s were:\n%s\n\tinstead of:\n%s\n", outPath, actual, expected)
	}
	if err = os.Remove(outPath); err != nil {
		t.Error(err)
	}
}

func TestStepToRecord(t *testing.T) {
	step := &model.Step{
		Component: model.StepComponent{
			Location: "/this/is/a/location",
			Name:     "test-name",
		},
		Config: model.StepConfig{
			Params: []model.StepParam{
				{
					Name:  "parameter-name",
					Value: "This is a test",
				},
			},
		},
	}
	actual := stepToRecord(step)
	expected := [][]string{
		{"", "parameter-name", "This is a test"},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Record %#v does not equal %#v", actual, expected)
	}
}

func TestWriteJobParameters(t *testing.T) {
	j := &model.Job{
		Steps: []model.Step{
			{
				Component: model.StepComponent{
					Location: "/this/is/a/location",
					Name:     "test-name",
				},
				Config: model.StepConfig{
					Params: []model.StepParam{
						{
							Name:  "parameter-name",
							Value: "This is a test",
						},
					},
				},
			},
		},
	}
	expected := `Executable,Argument Option,Argument Value
,parameter-name,This is a test
`
	if err := writeJobParameters("test", j); err != nil {
		t.Error(err)
	}
	outPath := "test/JobParameters.csv"
	input, err := ioutil.ReadFile(outPath)
	if err != nil {
		t.Error(err)
	}
	actual := string(input)
	if actual != expected {
		t.Errorf("Contents of %s were:\n%s\n\tinstead of:\n%s\n", outPath, actual, expected)
	}
	if err = os.Remove(outPath); err != nil {
		t.Error(err)
	}
}
