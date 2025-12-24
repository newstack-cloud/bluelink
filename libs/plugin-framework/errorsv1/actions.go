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
	PluginActionProviderListLinkTypes           = PluginAction("Provider::ListLinkTypes")
	PluginActionProviderListDataSourceTypes     = PluginAction("Provider::ListDataSourceTypes")
	PluginActionProviderListCustomVariableTypes = PluginAction("Provider::ListCustomVariableTypes")
	PluginActionProviderListFunctions           = PluginAction("Provider::ListFunctions")
	PluginActionProviderGetRetryPolicy          = PluginAction("Provider::GetRetryPolicy")

	PluginActionProviderCustomValidateResource        = PluginAction("Provider::CustomValidateResource")
	PluginActionProviderGetResourceSpecDefinition     = PluginAction("Provider::GetResourceSpecDefinition")
	PluginActionProviderCheckCanResourceLinkTo        = PluginAction("Provider::CheckCanResourceLinkTo")
	PluginActionProviderGetResourceStabilisedDeps     = PluginAction("Provider::GetResourceStabilisedDeps")
	PluginActionProviderCheckIsResourceCommonTerminal = PluginAction("Provider::CheckIsResourceCommonTerminal")
	PluginActionProviderGetResourceType               = PluginAction("Provider::GetResourceType")
	PluginActionProviderGetResourceExamples           = PluginAction("Provider::GetResourceExamples")
	PluginActionProviderGetResourceTypeDescription    = PluginAction("Provider::GetResourceTypeDescription")
	PluginActionProviderDeployResource                = PluginAction("Provider::DeployResource")
	PluginActionProviderCheckResourceHasStabilised    = PluginAction("Provider::CheckResourceHasStabilised")
	PluginActionProviderGetResourceExternalState      = PluginAction("Provider::GetResourceExternalState")
	PluginActionProviderDestroyResource               = PluginAction("Provider::DestroyResource")

	PluginActionProviderStageLinkChanges                    = PluginAction("Provider::StageLinkChanges")
	PluginActionProviderUpdateLinkResourceA                 = PluginAction("Provider::UpdateLinkResourceA")
	PluginActionProviderUpdateLinkResourceB                 = PluginAction("Provider::UpdateLinkResourceB")
	PluginActionProviderUpdateLinkIntermediaryResources     = PluginAction("Provider::UpdateLinkIntermediaryResources")
	PluginActionProviderGetLinkPriorityResource             = PluginAction("Provider::GetLinkPriorityResource")
	PluginActionProviderGetLinkTypeDescription              = PluginAction("Provider::GetLinkTypeDescription")
	PluginActionProviderGetLinkAnnotationDefinitions        = PluginAction("Provider::GetLinkAnnotationDefinitions")
	PluginActionProviderGetLinkKind                         = PluginAction("Provider::GetLinkKind")
	PluginActionProviderGetLinkIntermediaryExternalState    = PluginAction("Provider::GetLinkIntermediaryExternalState")

	PluginActionProviderGetDataSourceType            = PluginAction("Provider::GetDataSourceType")
	PluginActionProviderGetDataSourceTypeDescription = PluginAction("Provider::GetDataSourceTypeDescription")
	PluginActionProviderGetDataSourceExamples        = PluginAction("Provider::GetDataSourceExamples")
	PluginActionProviderCustomValidateDataSource     = PluginAction("Provider::CustomValidateDataSource")
	PluginActionProviderGetDataSourceSpecDefinition  = PluginAction("Provider::GetDataSourceSpecDefinition")
	PluginActionProviderGetDataSourceFilterFields    = PluginAction("Provider::GetDataSourceFilterFields")
	PluginActionProviderFetchDataSource              = PluginAction("Provider::FetchDataSource")

	PluginActionProviderGetCustomVariableType            = PluginAction("Provider::GetCustomVariableType")
	PluginActionProviderGetCustomVariableTypeDescription = PluginAction("Provider::GetCustomVariableTypeDescription")
	PluginActionProviderGetCustomVariableTypeOptions     = PluginAction("Provider::GetCustomVariableTypeOptions")
	PluginActionProviderGetCustomVariableTypeExamples    = PluginAction("Provider::GetCustomVariableTypeExamples")

	PluginActionProviderGetFunctionDefinition = PluginAction("Provider::GetFunctionDefinition")
	PluginActionProviderCallFunction          = PluginAction("Provider::CallFunction")

	///////////////////////////////////////////////////////////////////////////////////////
	// Transformer actions
	///////////////////////////////////////////////////////////////////////////////////////

	PluginActionTransformerGetTransformName          = PluginAction("Transformer::GetTransformName")
	PluginActionTransformerGetConfigDefinition       = PluginAction("Transformer::GetConfigDefinition")
	PluginActionTransformerTransform                 = PluginAction("Transformer::Transform")
	PluginActionTransformerListAbstractResourceTypes = PluginAction("Transformer::ListAbstractResourceTypes")

	PluginActionTransformerCustomValidateAbstractResource        = PluginAction("Transformer::CustomValidateAbstractResource")
	PluginActionTransformerGetAbstractResourceSpecDefinition     = PluginAction("Transformer::GetAbstractResourceSpecDefinition")
	PluginActionTransformerCheckCanAbstractResourceLinkTo        = PluginAction("Transformer::CheckCanAbstractResourceLinkTo")
	PluginActionTransformerCheckIsAbstractResourceCommonTerminal = PluginAction("Transformer::CheckIsAbstractResourceCommonTerminal")
	PluginActionTransformerGetAbstractResourceType               = PluginAction("Transformer::GetAbstractResourceType")
	PluginActionTransformerGetAbstractResourceExamples           = PluginAction("Transformer::GetAbstractResourceExamples")
	PluginActionTransformerGetAbstractResourceTypeDescription    = PluginAction("Transformer::GetAbstractResourceTypeDescription")

	///////////////////////////////////////////////////////////////////////////////////////
	// Service actions
	///////////////////////////////////////////////////////////////////////////////////////

	PluginActionServiceDeployResource        = PluginAction("Service::DeployResource")
	PluginActionServiceDestroyResource       = PluginAction("Service::DestroyResource")
	PluginActionServiceCallFunction          = PluginAction("Service::CallFunction")
	PluginActionServiceGetFunctionDefinition = PluginAction("Service::GetFunctionDefinition")
	PluginActionServiceCheckHasFunction      = PluginAction("Service::CheckHasFunction")
	PluginActionServiceListFunctions         = PluginAction("Service::ListFunctions")
	PluginActionServiceLookupResourceInState = PluginAction("Service::LookupResourceInState")
	PluginActionServiceAcquireResourceLock   = PluginAction("Service::AcquireResourceLock")
)
