package errorsv1

// PluginAction represents an action that a plugin
// or the plugin service can perform.
// This is primarily used in error handling.
type PluginAction string

const (
	///////////////////////////////////////////////////////////////////////////////////////
	// Provider actions
	///////////////////////////////////////////////////////////////////////////////////////

	PluginActionProviderGetNamespace            = PluginAction("Provider::GetNamespace")
	PluginActionProviderGetConfigDefinition     = PluginAction("Provider::GetConfigDefinition")
	PluginActionProviderListResourceTypes       = PluginAction("Provider::ListResourceTypes")
	PluginActionProviderListDataSourceTypes     = PluginAction("Provider::ListDataSourceTypes")
	PluginActionProviderListCustomVariableTypes = PluginAction("Provider::ListCustomVariableTypes")
	PluginActionProviderListFunctions           = PluginAction("Provider::ListFunctions")
	PluginActionProviderGetRetryPolicy          = PluginAction("Provider::GetRetryPolicy")

	PluginActionProviderCustomValidateResource        = PluginAction("Provider::CustomValidateResource")
	PluginActionProviderGetResourceSpecDefinition     = PluginAction("Provider::GetResourceSpecDefinition")
	PluginActionProviderCheckCanResourceLinkTo        = PluginAction("Provider::CheckCanResourceLinkTo")
	PluginActionProviderGetResourceStabilisedDeps     = PluginAction("Provider::GetResourceStabilisedDeps")
	PluginActionProviderCheckIsResourceCommonTerminal = PluginAction("Provider::CheckIsResourceCommonTerminal")
	PluginActionProviderGetResourceTypeDescription    = PluginAction("Provider::GetResourceTypeDescription")
	PluginActionProviderDeployResource                = PluginAction("Provider::DeployResource")
	PluginActionProviderCheckResourceHasStabilised    = PluginAction("Provider::CheckResourceHasStabilised")
	PluginActionProviderGetResourceExternalState      = PluginAction("Provider::GetResourceExternalState")
	PluginActionProviderDestroyResource               = PluginAction("Provider::DestroyResource")

	PluginActionProviderStageLinkChanges                = PluginAction("Provider::StageLinkChanges")
	PluginActionProviderUpdateLinkResourceA             = PluginAction("Provider::UpdateLinkResourceA")
	PluginActionProviderUpdateLinkResourceB             = PluginAction("Provider::UpdateLinkResourceB")
	PluginActionProviderUpdateLinkIntermediaryResources = PluginAction("Provider::UpdateLinkIntermediaryResources")
	PluginActionProviderGetLinkPriorityResource         = PluginAction("Provider::GetLinkPriorityResource")
	PluginActionProviderGetLinkKind                     = PluginAction("Provider::GetLinkKind")

	PluginActionProviderGetDataSourceTypeDescription = PluginAction("Provider::GetDataSourceTypeDescription")
	PluginActionProviderCustomValidateDataSource     = PluginAction("Provider::CustomValidateDataSource")
	PluginActionProviderGetDataSourceSpecDefinition  = PluginAction("Provider::GetDataSourceSpecDefinition")
	PluginActionProviderGetDataSourceFilterFields    = PluginAction("Provider::GetDataSourceFilterFields")
	PluginActionProviderFetchDataSource              = PluginAction("Provider::FetchDataSource")

	PluginActionProviderGetCustomVariableTypeDescription = PluginAction("Provider::GetCustomVariableTypeDescription")
	PluginActionProviderGetCustomVariableTypeOptions     = PluginAction("Provider::GetCustomVariableTypeOptions")

	PluginActionProviderGetFunctionDefinition = PluginAction("Provider::GetFunctionDefinition")
	PluginActionProviderCallFunction          = PluginAction("Provider::CallFunction")

	///////////////////////////////////////////////////////////////////////////////////////
	// Service actions
	///////////////////////////////////////////////////////////////////////////////////////

	PluginActionServiceDeployResource        = PluginAction("Service::DeployResource")
	PluginActionServiceDestroyResource       = PluginAction("Service::DestroyResource")
	PluginActionServiceCallFunction          = PluginAction("Service::CallFunction")
	PluginActionServiceGetFunctionDefinition = PluginAction("Service::GetFunctionDefinition")
	PluginActionServiceCheckHasFunction      = PluginAction("Service::CheckHasFunction")
	PluginActionServiceListFunctions         = PluginAction("Service::ListFunctions")
)
