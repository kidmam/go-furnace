package commands

import (
	"errors"
	"log"
	"reflect"
	"testing"

	commander "github.com/Yitsushi/go-commander"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
	"github.com/go-furnace/go-furnace/config"
	awsconfig "github.com/go-furnace/go-furnace/furnace-aws/config"
	"github.com/go-furnace/go-furnace/handle"
)

type fakeUpdateCFClient struct {
	cloudformationiface.CloudFormationAPI
	stackname string
	err       error
}

func init() {
	handle.LogFatalf = log.Fatalf
}

func (fc *fakeUpdateCFClient) ValidateTemplateRequest(input *cloudformation.ValidateTemplateInput) cloudformation.ValidateTemplateRequest {
	return cloudformation.ValidateTemplateRequest{
		Request: &aws.Request{
			Data:  &cloudformation.ValidateTemplateOutput{},
			Error: fc.err,
		},
		Input: input,
	}
}

func (fc *fakeUpdateCFClient) CreateChangeSetRequest(input *cloudformation.CreateChangeSetInput) cloudformation.CreateChangeSetRequest {
	return cloudformation.CreateChangeSetRequest{
		Request: &aws.Request{
			Data: &cloudformation.CreateChangeSetOutput{
				StackId: aws.String("DummyID"),
			},
			Error: fc.err,
		},
		Input: input,
	}
}

func (fc *fakeUpdateCFClient) ExecuteChangeSetRequest(input *cloudformation.ExecuteChangeSetInput) cloudformation.ExecuteChangeSetRequest {
	return cloudformation.ExecuteChangeSetRequest{
		Request: &aws.Request{
			Data:  &cloudformation.ExecuteChangeSetOutput{},
			Error: fc.err,
		},
		Input: input,
	}
}

func (fc *fakeUpdateCFClient) WaitUntilStackUpdateComplete(input *cloudformation.DescribeStacksInput) error {
	return nil
}

func (fc *fakeUpdateCFClient) WaitUntilChangeSetCreateComplete(input *cloudformation.DescribeChangeSetInput) error {
	return nil
}

func (fc *fakeUpdateCFClient) DescribeStacksRequest(input *cloudformation.DescribeStacksInput) cloudformation.DescribeStacksRequest {
	if fc.stackname == "NotEmptyStack" {
		return cloudformation.DescribeStacksRequest{
			Request: &aws.Request{
				Data:  &NotEmptyStack,
				Error: fc.err,
			},
		}
	}
	return cloudformation.DescribeStacksRequest{
		Request: &aws.Request{
			Data: &cloudformation.DescribeStacksOutput{},
		},
	}
}

// NotEmptyStack test structs which defines a non-empty stack.
var notEmptyChangeSetOutput = cloudformation.DescribeChangeSetOutput{
	ChangeSetId:   aws.String("id"),
	ChangeSetName: aws.String("name"),
	Changes: []cloudformation.Change{
		{
			ResourceChange: &cloudformation.ResourceChange{
				Action: cloudformation.ChangeAction("asdf"),
			},
		},
	},
	StackId:   aws.String("stackID"),
	StackName: aws.String("NotEmptyStack"),
}

func (fc *fakeUpdateCFClient) DescribeChangeSetRequest(input *cloudformation.DescribeChangeSetInput) cloudformation.DescribeChangeSetRequest {
	return cloudformation.DescribeChangeSetRequest{
		Request: &aws.Request{
			Data:  &notEmptyChangeSetOutput,
			Error: fc.err,
		},
	}
}

func TestUpdateExecute(t *testing.T) {
	config.WAITFREQUENCY = 0
	client := new(CFClient)
	stackname := "NotEmptyStack"
	client.Client = &fakeUpdateCFClient{err: nil, stackname: stackname}
	opts := &commander.CommandHelper{}
	opts.Args = append(opts.Args, "teststack")
	update(opts, client, true)
}

func TestUpdateExecuteWitCustomStack(t *testing.T) {
	config.WAITFREQUENCY = 0
	client := new(CFClient)
	stackname := "NotEmptyStack"
	client.Client = &fakeUpdateCFClient{err: nil, stackname: stackname}
	opts := &commander.CommandHelper{}
	opts.Args = append(opts.Args, "teststack")
	update(opts, client, true)
	if awsconfig.Config.Main.Stackname != "MyStack" {
		t.Fatal("test did not load the file requested.")
	}
}

func TestUpdateExecuteWitCustomStackNotFound(t *testing.T) {
	failed := false
	handle.LogFatalf = func(s string, a ...interface{}) {
		failed = true
	}
	config.WAITFREQUENCY = 0
	client := new(CFClient)
	stackname := "NotEmptyStack"
	client.Client = &fakeUpdateCFClient{err: nil, stackname: stackname}
	opts := &commander.CommandHelper{}
	opts.Args = append(opts.Args, "notfound")
	update(opts, client, true)
	if !failed {
		t.Error("Expected outcome to fail. Did not fail.")
	}
}

func TestUpdateExecuteEmptyStack(t *testing.T) {
	failed := false
	handle.LogFatalf = func(s string, a ...interface{}) {
		failed = true
	}
	config.WAITFREQUENCY = 0
	client := new(CFClient)
	stackname := "EmptyStack"
	client.Client = &fakeUpdateCFClient{err: nil, stackname: stackname}
	opts := &commander.CommandHelper{}
	opts.Args = append(opts.Args, "teststack")
	update(opts, client, true)
	if !failed {
		t.Error("Expected outcome to fail. Did not fail.")
	}
}

func TestUpdateProcedure(t *testing.T) {
	config.WAITFREQUENCY = 0
	client := new(CFClient)
	stackname := "NotEmptyStack"
	client.Client = &fakeUpdateCFClient{err: nil, stackname: stackname}
	awsconfig.Config = awsconfig.Configuration{}
	awsconfig.Config.Main.Stackname = "NotEmptyStack"
	opts := &commander.CommandHelper{}

	update(opts, client, true)
}

func TestUpdateStackReturnsWithError(t *testing.T) {
	failed := false
	expectedMessage := "failed to create stack"
	var message string
	handle.LogFatalf = func(s string, a ...interface{}) {
		failed = true
		message = a[0].(error).Error()
	}
	config.WAITFREQUENCY = 0
	client := new(CFClient)
	stackname := "NotEmptyStack"
	client.Client = &fakeUpdateCFClient{err: errors.New(expectedMessage), stackname: stackname}
	awsconfig.Config = awsconfig.Configuration{}
	awsconfig.Config.Main.Stackname = "NotEmptyStack"
	opts := &commander.CommandHelper{}
	update(opts, client, true)
	if !failed {
		t.Error("Expected outcome to fail. Did not fail.")
	}
	if message != expectedMessage {
		t.Errorf("message did not equal expected message of '%s', was:%s", expectedMessage, message)
	}
}

func TestUpdateCreate(t *testing.T) {
	wrapper := NewUpdate("furnace")
	if wrapper.Help.Arguments != "custom-config [-y]" ||
		!reflect.DeepEqual(wrapper.Help.Examples, []string{"", "custom-config", "-y", "mystack -y"}) ||
		wrapper.Help.LongDescription != `Update a stack with new parameters. -y can be given to automatically accept the applying of a changeset.` ||
		wrapper.Help.ShortDescription != "Update a stack" {
		t.Log(wrapper.Help.LongDescription)
		t.Log(wrapper.Help.ShortDescription)
		t.Log(wrapper.Help.Examples)
		t.Fatal("wrapper did not match with given params")
	}
}
