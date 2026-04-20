package schema

// GetDataSourceType safely extracts the type from a data source,
// returning an empty string if the type wrapper is nil or empty.
func GetDataSourceType(dataSource *DataSource) string {
	if dataSource.Type == nil || dataSource.Type.Value == "" {
		return ""
	}

	return dataSource.Type.Value
}

// GetResourceType safely extracts the type from a resource,
// returning an empty string if the type wrapper is nil or empty.
func GetResourceType(resource *Resource) string {
	if resource.Type == nil || resource.Type.Value == "" {
		return ""
	}

	return resource.Type.Value
}

// GetResourceRemovalPolicy safely extracts the removal policy value from a
// resource, returning an empty string if the wrapper or resource is nil or
// the value is unset.
// An empty value should be treated by callers as the default "delete" policy.
func GetResourceRemovalPolicy(resource *Resource) string {
	if resource == nil || resource.RemovalPolicy == nil {
		return ""
	}

	return string(resource.RemovalPolicy.Value)
}
