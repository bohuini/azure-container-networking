package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/fakes"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/cns/wireserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient is a mock implementation of HTTPClient
type MockHTTPClient struct {
	Response *http.Response
	Err      error
}

// Post is the implementation of the Post method for MockHTTPClient
func (m *MockHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	return m.Response, m.Err
}

func TestSendRegisterNodeRequest_StatusOK(t *testing.T) {
	ctx := context.Background()
	logger.InitLogger("testlogs", 0, 0, "./")
	httpServiceFake := fakes.NewHTTPServiceFake()
	nodeRegisterReq := cns.NodeRegisterRequest{
		NumCores:             2,
		NmAgentSupportedApis: nil,
	}

	url := "https://localhost:9000/api"

	// Create a mock HTTP client
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{"status": "success", "OrchestratorType": "Kubernetes", "DncPartitionKey": "1234", "NodeID": "5678"}`)),
		Header:     make(http.Header),
	}

	mockClient := &MockHTTPClient{Response: mockResponse, Err: nil}

	assert.NoError(t, sendRegisterNodeRequest(ctx, mockClient, httpServiceFake, nodeRegisterReq, url))
}

func TestSendRegisterNodeRequest_StatusAccepted(t *testing.T) {
	ctx := context.Background()
	logger.InitLogger("testlogs", 0, 0, "./")
	httpServiceFake := fakes.NewHTTPServiceFake()
	nodeRegisterReq := cns.NodeRegisterRequest{
		NumCores:             2,
		NmAgentSupportedApis: nil,
	}

	url := "https://localhost:9000/api"

	// Create a mock HTTP client
	mockResponse := &http.Response{
		StatusCode: http.StatusAccepted,
		Body:       io.NopCloser(bytes.NewBufferString(`{"status": "accepted", "OrchestratorType": "Kubernetes", "DncPartitionKey": "1234", "NodeID": "5678"}`)),
		Header:     make(http.Header),
	}

	mockClient := &MockHTTPClient{Response: mockResponse, Err: nil}

	assert.Error(t, sendRegisterNodeRequest(ctx, mockClient, httpServiceFake, nodeRegisterReq, url))
}

// Mock for wscliInterface
type mockWscli struct {
	mock.Mock
}

func (m *mockWscli) GetInterfaces(ctx context.Context) (*wireserver.GetInterfacesResult, error) {
	args := m.Called(ctx)
	return args.Get(0).(*wireserver.GetInterfacesResult), args.Error(1)
}

// Mock data
var (
	mockInterfaceInfo = &wireserver.InterfaceInfo{
		PrimaryIP: "10.0.0.1",
		Subnet:    "10.0.0.0/24",
		Gateway:   "10.0.0.254",
	}

	mockIPConfigStatus = cns.IPConfigurationStatus{
		NCID:      "test-nc",
		IPAddress: "10.0.0.2",
	}

	mockContainerStatus = map[string]ContainerStatus{
		"test-nc": {
			CreateNetworkContainerRequest: cns.CreateNetworkContainerRequest{
				IPConfiguration: cns.IPConfiguration{
					IPSubnet: cns.IPSubnet{
						PrefixLength: 24,
					},
				},
			},
		},
	}
)

func TestGetPrimaryNICMacAddress(t *testing.T) {
	// Create a mock wscli
	mockWscli := new(mockWscli)

	// Set up the state
	state := &HTTPRestServiceState{
		ContainerStatus: mockContainerStatus,
	}

	// Test success case
	mockWscli.On("GetInterfaces", mock.Anything).Return(&wireserver.GetInterfacesResult{
		Interface: *mockInterfaceInfo,
	}, nil)

	macAddress, err := GetPrimaryNICMacAddress(state, mockIPConfigStatus, mockWscli)
	assert.NoError(t, err)
	assert.Equal(t, mockInterfaceInfo.MacAddress, macAddress)

	// Test error case: No container status
	state.ContainerStatus = map[string]ContainerStatus{}
	_, err = GetPrimaryNICMacAddress(state, mockIPConfigStatus, mockWscli)
	assert.Error(t, err)
	assert.Equal(t, "Failed to get NC Configuration for NcId: test-nc", err.Error())
}
