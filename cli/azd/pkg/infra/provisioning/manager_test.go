// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provisioning_test

import (
	"context"
	"strings"
	"testing"

	"github.com/azure/azure-dev/cli/azd/pkg/account"
	"github.com/azure/azure-dev/cli/azd/pkg/environment"
	"github.com/azure/azure-dev/cli/azd/pkg/infra/provisioning"
	. "github.com/azure/azure-dev/cli/azd/pkg/infra/provisioning"
	"github.com/azure/azure-dev/cli/azd/pkg/infra/provisioning/test"
	"github.com/azure/azure-dev/cli/azd/pkg/input"
	"github.com/azure/azure-dev/cli/azd/pkg/prompt"
	"github.com/azure/azure-dev/cli/azd/pkg/tools/azcli"
	"github.com/azure/azure-dev/cli/azd/test/mocks"
	"github.com/azure/azure-dev/cli/azd/test/mocks/mockaccount"
	"github.com/azure/azure-dev/cli/azd/test/mocks/mockazcli"
	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

func TestProvisionInitializesEnvironment(t *testing.T) {
	env := environment.EphemeralWithValues("test-env", nil)

	mockContext := mocks.NewMockContext(context.Background())
	mockContext.Console.WhenSelect(func(options input.ConsoleOptions) bool {
		return strings.Contains(options.Message, "Select an Azure Subscription to use")
	}).RespondFn(func(options input.ConsoleOptions) (any, error) {
		// Select the first from the list
		return 0, nil
	})
	mockContext.Console.WhenSelect(func(options input.ConsoleOptions) bool {
		return strings.Contains(options.Message, "Select an Azure location")
	}).RespondFn(func(options input.ConsoleOptions) (any, error) {
		// Select the first from the list
		return 0, nil
	})

	registerContainerDependencies(mockContext, env)

	mgr := NewManager(mockContext.Container, env, mockContext.Console, mockContext.AlphaFeaturesManager, nil)
	err := mgr.Initialize(*mockContext.Context, "", Options{Provider: "test"})
	require.NoError(t, err)

	require.Equal(t, "00000000-0000-0000-0000-000000000000", env.GetSubscriptionId())
	require.Equal(t, "location", env.GetLocation())
}

func TestManagerPreview(t *testing.T) {
	env := environment.EphemeralWithValues("test-env", map[string]string{
		"AZURE_SUBSCRIPTION_ID": "SUBSCRIPTION_ID",
		"AZURE_LOCATION":        "eastus2",
	})

	mockContext := mocks.NewMockContext(context.Background())
	registerContainerDependencies(mockContext, env)

	mgr := NewManager(mockContext.Container, env, mockContext.Console, mockContext.AlphaFeaturesManager, nil)
	err := mgr.Initialize(*mockContext.Context, "", Options{Provider: "test"})
	require.NoError(t, err)

	deploymentPlan, err := mgr.Preview(*mockContext.Context)

	require.NotNil(t, deploymentPlan)
	require.Nil(t, err)
}

func TestManagerGetState(t *testing.T) {
	env := environment.EphemeralWithValues("test-env", map[string]string{
		"AZURE_SUBSCRIPTION_ID": "SUBSCRIPTION_ID",
		"AZURE_LOCATION":        "eastus2",
	})

	mockContext := mocks.NewMockContext(context.Background())
	registerContainerDependencies(mockContext, env)

	mgr := NewManager(mockContext.Container, env, mockContext.Console, mockContext.AlphaFeaturesManager, nil)
	err := mgr.Initialize(*mockContext.Context, "", Options{Provider: "test"})
	require.NoError(t, err)

	getResult, err := mgr.State(*mockContext.Context)

	require.NotNil(t, getResult)
	require.Nil(t, err)
}

func TestManagerDeploy(t *testing.T) {
	env := environment.EphemeralWithValues("test-env", map[string]string{
		"AZURE_SUBSCRIPTION_ID": "SUBSCRIPTION_ID",
		"AZURE_LOCATION":        "eastus2",
	})

	mockContext := mocks.NewMockContext(context.Background())
	registerContainerDependencies(mockContext, env)

	mgr := NewManager(mockContext.Container, env, mockContext.Console, mockContext.AlphaFeaturesManager, nil)
	err := mgr.Initialize(*mockContext.Context, "", Options{Provider: "test"})
	require.NoError(t, err)

	deployResult, err := mgr.Deploy(*mockContext.Context)

	require.NotNil(t, deployResult)
	require.Nil(t, err)
}

func TestManagerDestroyWithPositiveConfirmation(t *testing.T) {
	env := environment.EphemeralWithValues("test-env", map[string]string{
		"AZURE_SUBSCRIPTION_ID": "SUBSCRIPTION_ID",
		"AZURE_LOCATION":        "eastus2",
	})

	mockContext := mocks.NewMockContext(context.Background())
	mockContext.Console.WhenConfirm(func(options input.ConsoleOptions) bool {
		return strings.Contains(options.Message, "Are you sure you want to destroy?")
	}).Respond(true)

	registerContainerDependencies(mockContext, env)

	mgr := NewManager(mockContext.Container, env, mockContext.Console, mockContext.AlphaFeaturesManager, nil)
	err := mgr.Initialize(*mockContext.Context, "", Options{Provider: "test"})
	require.NoError(t, err)

	destroyOptions := NewDestroyOptions(false, false)
	destroyResult, err := mgr.Destroy(*mockContext.Context, destroyOptions)

	require.NotNil(t, destroyResult)
	require.Nil(t, err)
	require.Contains(t, mockContext.Console.Output(), "Are you sure you want to destroy?")
}

func TestManagerDestroyWithNegativeConfirmation(t *testing.T) {
	env := environment.EphemeralWithValues("test-env", map[string]string{
		"AZURE_SUBSCRIPTION_ID": "SUBSCRIPTION_ID",
		"AZURE_LOCATION":        "eastus2",
	})

	mockContext := mocks.NewMockContext(context.Background())

	mockContext.Console.WhenConfirm(func(options input.ConsoleOptions) bool {
		return strings.Contains(options.Message, "Are you sure you want to destroy?")
	}).Respond(false)

	registerContainerDependencies(mockContext, env)

	mgr := NewManager(mockContext.Container, env, mockContext.Console, mockContext.AlphaFeaturesManager, nil)
	err := mgr.Initialize(*mockContext.Context, "", Options{Provider: "test"})
	require.NoError(t, err)

	destroyOptions := NewDestroyOptions(false, false)
	destroyResult, err := mgr.Destroy(*mockContext.Context, destroyOptions)

	require.Nil(t, destroyResult)
	require.NotNil(t, err)
	require.Contains(t, mockContext.Console.Output(), "Are you sure you want to destroy?")
}

func registerContainerDependencies(mockContext *mocks.MockContext, env *environment.Environment) {
	mockContext.Container.RegisterSingleton(prompt.NewDefaultPrompter)
	_ = mockContext.Container.RegisterNamedTransient(string(provisioning.Test), test.NewTestProvider)
	mockContext.Container.RegisterSingleton(func() account.Manager {
		return &mockaccount.MockAccountManager{
			Subscriptions: []account.Subscription{
				{
					Id:   "00000000-0000-0000-0000-000000000000",
					Name: "test",
				},
			},
			Locations: []account.Location{
				{
					Name:                "location",
					DisplayName:         "Test Location",
					RegionalDisplayName: "(US) Test Location",
				},
			},
		}
	})
	mockContext.Container.RegisterSingleton(func() *environment.Environment {
		return env
	})
	mockContext.Container.RegisterSingleton(func() azcli.AzCli {
		return mockazcli.NewAzCliFromMockContext(mockContext)
	})

	mockContext.Container.RegisterSingleton(func() clock.Clock {
		return clock.NewMock()
	})
}
