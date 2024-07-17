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
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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

// Mock implementation of wscliInterface
type mockWscli struct {
	result *wireserver.GetInterfacesResult
	err    error
}

func (m *mockWscli) GetInterfaces(ctx context.Context) (*wireserver.GetInterfacesResult, error) {
	return m.result, m.err
}

func TestGetPrimaryNICMACAddress(t *testing.T) {
	tests := []struct {
		name     string
		wscli    wscliInterface
		expected string
		wantErr  bool
	}{
		{
			name: "Primary interface found",
			wscli: &mockWscli{
				result: &wireserver.GetInterfacesResult{
					Interface: []wireserver.Interface{
						{
							MacAddress: "00-11-22-33-44-55",
							IsPrimary:  true,
							IPSubnet: []wireserver.Subnet{
								{
									Prefix: "192.168.1.0/24",
								},
							},
						},
					},
				},
				err: nil,
			},
			expected: "00-11-22-33-44-55",
			wantErr:  false,
		},
		{
			name: "No primary interface",
			wscli: &mockWscli{
				result: &wireserver.GetInterfacesResult{
					Interface: []wireserver.Interface{
						{
							MacAddress: "00-11-22-33-44-55",
							IsPrimary:  false,
							IPSubnet: []wireserver.Subnet{
								{
									Prefix: "192.168.1.0/24",
								},
							},
						},
					},
				},
				err: nil,
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "No subnets in primary interface",
			wscli: &mockWscli{
				result: &wireserver.GetInterfacesResult{
					Interface: []wireserver.Interface{
						{
							MacAddress: "00-11-22-33-44-55",
							IsPrimary:  true,
							IPSubnet:   []wireserver.Subnet{},
						},
					},
				},
				err: nil,
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "Error fetching interfaces",
			wscli: &mockWscli{
				result: nil,
				err:    errors.New("failed to get interfaces"),
			},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			macAddress, err := getPrimaryNICMACAddress(tt.wscli)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, macAddress)
		})
	}
}
