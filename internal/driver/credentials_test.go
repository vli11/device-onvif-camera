// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2022 Intel Corporation
// Copyright (c) 2023 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"sync"
	"testing"

	"github.com/IOTechSystems/onvif"
	"github.com/edgexfoundry/go-mod-bootstrap/v3/bootstrap/interfaces/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsAuthModeValid verifies auth mode is set correctly.
func TestIsAuthModeValid(t *testing.T) {

	tests := []struct {
		input    string
		expected bool
	}{
		{
			input:    onvif.DigestAuth,
			expected: true,
		},
		{
			input:    onvif.UsernameTokenAuth,
			expected: true,
		},
		{
			input:    onvif.Both,
			expected: true,
		},
		{
			input:    onvif.NoAuth,
			expected: true,
		},
		{
			input:    "invalidValue",
			expected: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			result := IsAuthModeValid(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestTryGetCredentials verifies correct credentials are returned.
func TestTryGetCredentials(t *testing.T) {

	tests := []struct {
		secretName    string
		expected      Credentials
		errorExpected bool
		mockUsername  string
		mockPassword  string
		mockAuthMode  string
	}{
		{
			secretName:   noAuthSecretName,
			mockUsername: "username",
			mockPassword: "password",
			mockAuthMode: onvif.DigestAuth,
			expected: Credentials{
				AuthMode: AuthModeNone,
			},
		},
		{
			secretName:   "validSecretName",
			mockUsername: "username",
			mockPassword: "password",
			mockAuthMode: onvif.DigestAuth,
			expected: Credentials{
				AuthMode: AuthModeDigest,
				Username: "username",
				Password: "password",
			},
		},
		{
			secretName:    "invalidSecretName",
			mockUsername:  "username",
			mockPassword:  "password",
			mockAuthMode:  onvif.DigestAuth,
			errorExpected: true,
		},
		{
			secretName:   "validSecretNameInvalidAuthMode",
			mockUsername: "username",
			mockPassword: "password",
			mockAuthMode: "invalidAuthMode",
			expected: Credentials{
				AuthMode: AuthModeUsernameToken,
				Username: "username",
				Password: "password",
			},
		},
	}

	driver, mockService := createDriverWithMockService()

	mockSecretProvider := &mocks.SecretProvider{}
	mockService.On("SecretProvider").Return(mockSecretProvider)

	for _, test := range tests {
		test := test
		t.Run(test.secretName, func(t *testing.T) {
			if test.errorExpected {
				mockSecretProvider.On("GetSecret", test.secretName, UsernameKey, PasswordKey, AuthModeKey).Return(nil, errors.NewCommonEdgeX(errors.KindServerError, "unit test error", nil)).Once()
			} else {
				mockSecretProvider.On("GetSecret", test.secretName, UsernameKey, PasswordKey, AuthModeKey).Return(map[string]string{"username": test.mockUsername, "password": test.mockPassword, "mode": test.mockAuthMode}, nil).Once()
			}
			actual, err := driver.tryGetCredentials(test.secretName)

			if test.errorExpected {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected.Username, actual.Username)
			assert.Equal(t, test.expected.Password, actual.Password)
			assert.Equal(t, test.expected.AuthMode, actual.AuthMode)
		})
	}
}

// TestTryGetCredentialsForDevice verifies correct credentials are returned for a device based on the MAC address of the device.
func TestTryGetCredentialsForDevice(t *testing.T) {

	tests := []struct {
		existingProtocols map[string]models.ProtocolProperties
		device            models.Device
		expected          Credentials
		secretName        string

		errorExpected bool
		username      string
		password      string
		authMode      string
	}{
		{
			existingProtocols: map[string]models.ProtocolProperties{
				OnvifProtocol: {
					MACAddress: "",
				},
			},

			secretName:    "default_secret_name",
			username:      "username",
			password:      "password",
			authMode:      onvif.DigestAuth,
			errorExpected: true,
		},
		{
			existingProtocols: map[string]models.ProtocolProperties{
				OnvifProtocol: {
					MACAddress: "aa:bb:cc:dd:ee:ff",
				},
			},

			secretName: "secret_name",
			username:   "username",
			password:   "password",
			authMode:   onvif.UsernameTokenAuth,
			expected: Credentials{
				AuthMode: AuthModeUsernameToken,
				Username: "username",
				Password: "password",
			},
		},
	}

	driver, mockService := createDriverWithMockService()

	driver.macAddressMapper = NewMACAddressMapper(mockService)
	driver.macAddressMapper.credsMap = convertMACMappings(t, map[string]string{
		"secret_name": "aa:bb:cc:dd:ee:ff",
	})
	driver.configMu = new(sync.RWMutex)
	driver.config = &ServiceConfig{
		AppCustom: CustomConfig{
			DefaultSecretName: "default_secret_name",
		},
	}

	mockSecretProvider := &mocks.SecretProvider{}

	for i := range tests {
		if tests[i].errorExpected {
			mockSecretProvider.On("GetSecret", tests[i].secretName, UsernameKey, PasswordKey, AuthModeKey).Return(nil, errors.NewCommonEdgeX(errors.KindServerError, "unit test error", nil)).Once()
		} else {
			mockSecretProvider.On("GetSecret", tests[i].secretName, UsernameKey, PasswordKey, AuthModeKey).Return(map[string]string{"username": tests[i].username, "password": tests[i].password, "mode": tests[i].authMode}, nil).Once()
		}
	}

	mockService.On("SecretProvider").Return(mockSecretProvider)

	for _, test := range tests {
		test := test
		t.Run(test.secretName, func(t *testing.T) {

			actual, err := driver.tryGetCredentialsForDevice(createTestDeviceWithProtocols(test.existingProtocols))

			if test.errorExpected {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected.Username, actual.Username)
			assert.Equal(t, test.expected.Password, actual.Password)
			assert.Equal(t, test.expected.AuthMode, actual.AuthMode)
		})
	}
}
