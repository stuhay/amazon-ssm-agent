// Copyright 2016 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the
// License is located at
//
// http://aws.amazon.com/apache2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

// Package copycontent implements the aws:copyContent plugin
package copycontent

import (
	"testing"

	"github.com/aws/amazon-ssm-agent/agent/contracts"
	filemock "github.com/aws/amazon-ssm-agent/agent/fileutil/filemanager/mock"
	"github.com/aws/amazon-ssm-agent/agent/log"
	"github.com/aws/amazon-ssm-agent/agent/plugins/copycontent/remoteresource"
	resourcemock "github.com/aws/amazon-ssm-agent/agent/plugins/copycontent/remoteresource/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var logger = log.NewMockLog()
var copyContentFileMock = filemock.FileSystemMock{}
var copyContentResourceMock = resourcemock.RemoteResourceMock{}

func TestNewRemoteResource_InvalidLocationType(t *testing.T) {

	var mockLocationInfo string
	remoteresource, err := newRemoteResource(logger, "invalid", mockLocationInfo)

	assert.Nil(t, remoteresource)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Location type")

}

func TestNewRemoteResource_Github(t *testing.T) {

	locationInfo := `{
		"owner" : "test-owner",
		"repository" :	 "test-repo"
		}`
	remoteresource, err := newRemoteResource(logger, "GitHub", locationInfo)

	assert.NotNil(t, remoteresource)
	assert.NoError(t, err)

}

func TestNewRemoteResource_S3(t *testing.T) {

	locationInfo := `{
		"path" : "https://s3.amazonaws.com/test-bucket/fake-key/"
		}`
	remoteresource, err := newRemoteResource(logger, "S3", locationInfo)

	assert.NotNil(t, remoteresource)
	assert.NoError(t, err)

}

func TestNewRemoteResource_SSMDocument(t *testing.T) {

	locationInfo := `{
		"name" : "doc-name",
		"version" : "1"
		}`
	remoteresource, err := newRemoteResource(logger, "SSMDocument", locationInfo)

	assert.NotNil(t, remoteresource)
	assert.NoError(t, err)

}

func TestNewPlugin_RunCopyContent(t *testing.T) {

	fileMock := filemock.FileSystemMock{}

	input := CopyContentPlugin{
		LocationType:   "Github",
		DestinationDir: "destination",
	}
	config := createStubConfiguration("orch", "bucket", "prefix", "1234-1234-1234", "directory")

	p := Plugin{
		remoteResourceCreator: fakeRemoteResource,
		filesys:               fileMock,
	}
	output := contracts.PluginOutput{}

	p.runCopyContent(logger, &input, config, &output)

	assert.Equal(t, output.Status, contracts.ResultStatusSuccess)
	copyContentResourceMock.AssertExpectations(t)
	fileMock.AssertExpectations(t)
}

func TestNewPlugin_RunCopyContent_absPathDestinationDir(t *testing.T) {

	fileMock := filemock.FileSystemMock{}

	input := CopyContentPlugin{
		LocationType:   "Github",
		DestinationDir: "/var/temp/fake-dir",
	}
	config := createStubConfiguration("orch", "bucket", "prefix", "1234-1234-1234", "directory")

	p := Plugin{
		remoteResourceCreator: absoluteDestinationDirRemoteResource,
		filesys:               fileMock,
	}
	output := contracts.PluginOutput{}

	p.runCopyContent(logger, &input, config, &output)

	assert.Equal(t, output.Status, contracts.ResultStatusSuccess)
	copyContentResourceMock.AssertExpectations(t)
	fileMock.AssertExpectations(t)
}

func Test_RunCopyContentBadLocationInfo(t *testing.T) {

	fileMock := filemock.FileSystemMock{}
	locationInfo := `{
		"owner" = "test-owner",
		"repository" = "test-repo"
		}`

	config := createStubConfiguration("orch", "bucket", "prefix", "1234-1234-1234", "directory")

	input := CopyContentPlugin{
		LocationType:   "GitHub",
		LocationInfo:   locationInfo,
		DestinationDir: "",
	}
	p := Plugin{
		remoteResourceCreator: newRemoteResource,
		filesys:               fileMock,
	}
	output := contracts.PluginOutput{}
	p.runCopyContent(logger, &input, config, &output)

	assert.Equal(t, output.Status, contracts.ResultStatusFailed)
	fileMock.AssertExpectations(t)

}

func TestValidateInput_UnsupportedLocationType(t *testing.T) {

	input := CopyContentPlugin{}
	input.LocationType = "unknown"

	validateInput(&input)

	result, err := validateInput(&input)

	assert.False(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported location type")
}

func TestValidateInput_UnknownLocationType(t *testing.T) {

	input := CopyContentPlugin{}

	validateInput(&input)

	result, err := validateInput(&input)

	assert.False(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Location Type must be specified")
}

func TestValidateInput_NoLocationInfo(t *testing.T) {

	input := CopyContentPlugin{}
	input.LocationType = "S3"

	validateInput(&input)

	result, err := validateInput(&input)

	assert.False(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Location Information must be specified")
}

func TestName(t *testing.T) {
	assert.Equal(t, "aws:copyContent", Name())
}

func TestParseAndValidateInput_NoInput(t *testing.T) {
	rawPluginInput := ""

	_, err := parseAndValidateInput(rawPluginInput)

	assert.Error(t, err)
}

// Mock and stub functions
func fakeRemoteResource(log log.T, locationType string, locationInfo string) (remoteresource.RemoteResource, error) {

	copyContentResourceMock.On("ValidateLocationInfo").Return(true, nil).Once()
	copyContentResourceMock.On("Download", logger, copyContentFileMock, mock.Anything).Return(nil).Once()
	return copyContentResourceMock, nil
}

func absoluteDestinationDirRemoteResource(log log.T, locationType string, locationInfo string) (remoteresource.RemoteResource, error) {

	copyContentResourceMock.On("ValidateLocationInfo").Return(true, nil).Once()
	copyContentResourceMock.On("Download", logger, copyContentFileMock, "/var/temp/fake-dir").Return(nil).Once()
	return copyContentResourceMock, nil
}

func createStubConfiguration(orch, bucket, prefix, message, dir string) contracts.Configuration {
	return contracts.Configuration{
		OrchestrationDirectory:  orch,
		OutputS3BucketName:      bucket,
		OutputS3KeyPrefix:       prefix,
		MessageId:               message,
		PluginID:                "aws-copyContent",
		DefaultWorkingDirectory: dir,
	}
}