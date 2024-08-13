//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

package generated

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

func (client *BlockBlobClient) Endpoint() string {
	return client.endpoint
}

func (client *BlockBlobClient) Internal() *azcore.Client {
	return client.internal
}

// NewBlockBlobClient creates a new instance of BlockBlobClient with the specified values.
//   - endpoint - The URL of the service account, container, or blob that is the target of the desired operation.
//   - azClient - azcore.Client is a basic HTTP client. It consists of a pipeline and tracing provider.
func NewBlockBlobClient(endpoint string, azClient *azcore.Client) *BlockBlobClient {
	client := &BlockBlobClient{
		internal: azClient,
		endpoint: endpoint,
	}
	return client
}
