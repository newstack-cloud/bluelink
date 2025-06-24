package substitutions

const (
	// SubstitutionFunctionFromJSON is a function that is used to extract values
	// from a serialised JSON string.
	SubstitutionFunctionFromJSON SubstitutionFunctionName = "fromjson"

	// SubstitutionFunctionFromJSON_G is a higher-order function that creates a function
	// that is used to extract values from a serialised JSON string.
	// Example:
	// ${map(variables.cacheClusterConfigDefs, fromjson_g("host"))}
	SubstitutionFunctionFromJSON_G SubstitutionFunctionName = "fromjson_g"

	// SubstitutionFunctionJSONDecode is a function that is used to decode a serialised json string
	// into an array or mapping.
	SubstitutionFunctionJSONDecode SubstitutionFunctionName = "jsondecode"

	// SubstitutionFunctionLen is a function that is used to get the length of a string, array
	// or mapping.
	SubstitutionFunctionLen SubstitutionFunctionName = "len"

	// SubstitutionFunctionSubstr is a function that is used to get a substring of a given string.
	SubstitutionFunctionSubstr SubstitutionFunctionName = "substr"

	// SubstitutionFunctionSubstr_G is a higher-order function that creates a function
	// that is used to get a substring from a given string.
	// Example:
	// ${map(variables.cacheClusterConfig.hosts, substr_g(0, 3))}
	SubstitutionFunctionSubstr_G SubstitutionFunctionName = "substr_g"

	// Replace is a function that is used to replace all occurrences
	// of a given string with another string.
	SubstitutionFunctionReplace SubstitutionFunctionName = "replace"

	// SubstitutionFunctionReplace_G is a higher-order function that creates a function
	// that is used to replace all occurences of a given string with another string.
	// Example:
	// ${map(variables.cacheClusterConfig.hosts, replace_g("http://", "https://"))}
	SubstitutionFunctionReplace_G SubstitutionFunctionName = "replace_g"

	// SubstitutionFunctionTrim is a function that is used to remove all leading and trailing whitespace
	// from a given string.
	SubstitutionFunctionTrim SubstitutionFunctionName = "trim"

	// SubstitutionFunctionTrimPrefix is a function that is used to remove a prefix from a string.
	SubstitutionFunctionTrimPrefix SubstitutionFunctionName = "trimprefix"

	// SubstitutionFunctionTrimPrefix_G is a higher-order function that creates a function
	// that is used to remove a prefix from a string.
	// Example:
	// ${map(variables,cacheClusterConfig.hosts, trimprefix_g("http://"))}
	SubstitutionFunctionTrimPrefix_G SubstitutionFunctionName = "trimprefix_g"

	// SubstitutionFunctionTrimSuffix is a function that is used to remove a suffix from a string.
	SubstitutionFunctionTrimSuffix SubstitutionFunctionName = "trimsuffix"

	// SubstitutionFunctionTrimSuffix_G is a higher-order function that creates a function
	// that is used to remove a suffix from a string.
	// Example:
	// ${map(variables.cacheClusterConfig.hosts, trimsuffix_g("/config"))}
	SubstitutionFunctionTrimSuffix_G SubstitutionFunctionName = "trimsuffix_g"

	// SubstitutionFunctionSplit is a function that is used to split a string
	// into an array of strings based on a delimiter.
	SubstitutionFunctionSplit SubstitutionFunctionName = "split"

	// SubstitutionFunctionSplit_G is a higher-order function that creates a function
	// that is used to split a string into an array of strings based on a delimiter.
	// Example:
	// ${flatmap(variables.cacheClusterConfig.multiClusterHosts, split_g(","))}
	SubstitutionFunctionSplit_G SubstitutionFunctionName = "split_g"

	// SubstitutionFunctionJoin is a function that is used to join an array of strings
	// into a single string with a delimiter.
	SubstitutionFunctionJoin SubstitutionFunctionName = "join"

	// SubstitutionFunctionIndex is a function that is used to get the
	// first index of a substring in a given string.
	SubstitutionFunctionIndex SubstitutionFunctionName = "index"

	// SubstitutionFunctionLastIndex is a function that is used to get the
	// last index of a substring in a given string.
	SubstitutionFunctionLastIndex SubstitutionFunctionName = "last_index"

	// SubstitutionFunctionToUpper is a function that converts all characters
	// of a string to upper case.
	SubstitutionFunctionToUpper SubstitutionFunctionName = "to_upper"

	// SubstitutionFunctionToLower is a function that converts all characters
	// of a string to lower case.
	SubstitutionFunctionToLower SubstitutionFunctionName = "to_lower"

	// SubstitutionFunctionHasPrefix is a function that checks if a string
	// starts with a given substring.
	SubstitutionFunctionHasPrefix SubstitutionFunctionName = "has_prefix"

	// SubstitutionFunctionHasPrefix_G is a higher-order function that creates a function
	// that is used to check if a string starts with a given substring.
	// Example:
	// ${filter(
	// 	variables.cacheClusterConfig.hosts,
	// 	has_prefix_g("http://")
	// )}
	SubstitutionFunctionHasPrefix_G SubstitutionFunctionName = "has_prefix_g"

	// SubstitutionFunctionHasSuffix is a function that checks if a string
	// ends with a given substring.
	SubstitutionFunctionHasSuffix SubstitutionFunctionName = "has_suffix"

	// SubstitutionFunctionHasSuffix_G is a higher-order function that creates a function
	// that is used to check if a string ends with a given substring.
	// Example:
	// ${filter(
	// 	variables.cacheClusterConfig.hosts,
	// 	has_suffix_g("/config")
	// )}
	SubstitutionFunctionHasSuffix_G SubstitutionFunctionName = "has_suffix_g"

	// SubstitutionFunctionContains is a function that checks if a string
	// contains a given substring or an array contains a given element.
	SubstitutionFunctionContains SubstitutionFunctionName = "contains"

	// SubstitutionFunctionContains_G is a higher-order function that creates a function
	// that is used to check if a string contains a given substring
	// or an array contains a given element.
	// Example:
	// ${filter(
	// 	variables.cacheClusterConfig.hosts,
	// 	contains_g("blueprint.dev")
	// )}
	SubstitutionFunctionContains_G SubstitutionFunctionName = "contains_g"

	// SubstitutionFunctionList is a function that creates an array
	// from variadic arguments of the same type.
	SubstitutionFunctionList SubstitutionFunctionName = "list"

	// SubstitutionFunctionObject is a function that creates an object
	// from variadic named arguments.
	SubstitutionFunctionObject SubstitutionFunctionName = "object"

	// SubstitutionFunctionKeys is a function that produces an array of keys
	// from a mapping or attribute names from an object.
	SubstitutionFunctionKeys SubstitutionFunctionName = "keys"

	// SubstitutionFunctionVals is a function that produces an array of values
	// from a mapping.
	SubstitutionFunctionVals SubstitutionFunctionName = "vals"

	// SubstitutionFunctionMap is a function that maps a list of values
	// to a new list of values using a function.
	SubstitutionFunctionMap SubstitutionFunctionName = "map"

	// SubstitutionFunctionFilter is a function that filters a list of values
	// based on a predicate function.
	SubstitutionFunctionFilter SubstitutionFunctionName = "filter"

	// SubstitutionFunctionReduce is a function that reduces a list of values
	// to a single value using a function.
	SubstitutionFunctionReduce SubstitutionFunctionName = "reduce"

	// SubstitutionFunctionSort is a function that sorts a list of values
	// to a single value using a comparison function.
	SubstitutionFunctionSort SubstitutionFunctionName = "sort"

	// SubstitutionFunctionFlatMap is a function that maps a list of values
	// using a function and flattens the result.
	SubstitutionFunctionFlatMap SubstitutionFunctionName = "flatmap"

	// SubstitutionFunctionCompose is a higher-order function that combines
	// N functions into a single function, where the output of one function
	// is passed in as the input of the previous function. The call order of the function
	// is from right to left.
	SubstitutionFunctionCompose SubstitutionFunctionName = "compose"

	// SubstitutionFunctionPipe is a higher-order function that combines
	// N functions into a single function, where the output of one function
	// is passed in as the input of the next function. The call order of the function
	// is from left to right.
	SubstitutionFunctionPipe SubstitutionFunctionName = "pipe"

	// SubstitutionFunctionGetAttr is a higher-order function that returns
	// a function that extracts a named attribute from an object or a mapping.
	// This is useful in situations where you want to map an array of objects to an array
	// of values of a specific attribute such as IDs.
	SubstitutionFunctionGetAttr SubstitutionFunctionName = "getattr"

	// SubstitutionFunctionGetElem is a higher-order function that returns
	// a function that extracts an element from an array.
	// This is useful when you want to map a two-dimensional array to an array
	// of values of a specific element.
	SubstitutionFunctionGetElem SubstitutionFunctionName = "getelem"

	// SubstitutionFunctionLink is a function that is used to retrieve the state
	// of a link between two resources in the current blueprint.
	SubstitutionFunctionLink SubstitutionFunctionName = "link"

	// SubstitutionFunctionAnd is a function that is used to perform a logical AND
	// operation on two boolean values.
	SubstitutionFunctionAnd SubstitutionFunctionName = "and"

	// SubstitutionFunctionOr is a function that is used to perform a logical OR
	// operation on two boolean values.
	SubstitutionFunctionOr SubstitutionFunctionName = "or"

	// SubstitutionFunctionNot is a function that is used to perform a negation
	// on a boolean value.
	SubstitutionFunctionNot SubstitutionFunctionName = "not"

	// SubstitutionFunctionEq is a function that is used to perform an equality
	// comparison on two values.
	SubstitutionFunctionEq SubstitutionFunctionName = "eq"

	// substitutionFunctionGT is a function that is used to perform a greater than
	// comparison on two values.
	SubstitutionFunctionGT SubstitutionFunctionName = "gt"

	// SubstitutionFunctionGE is a function that is used to perform a greater than
	// or equal to comparison on two values.
	SubstitutionFunctionGE SubstitutionFunctionName = "ge"

	// SubstitutionFunctionLT is a function that is used to perform a less than
	// comparison on two values.
	SubstitutionFunctionLT SubstitutionFunctionName = "lt"

	// SubstitutionFunctionLE is a function that is used to perform a less than
	// or equal to comparison on two values.
	SubstitutionFunctionLE SubstitutionFunctionName = "le"

	// SubstituionFunctionCWD is a function that is used to get the current working
	// directory of the user executing or validating a blueprint.
	SubstitutionFunctionCWD SubstitutionFunctionName = "cwd"

	// SubstitutionFunctionDateTime is a function that is used to get the current
	// date and time in a specific format.
	SubstitutionFunctionDateTime SubstitutionFunctionName = "datetime"
)

var (
	// CoreSubstitutionFunctions provides a slice of all the core
	// functions that can be called in a substitution within ${..}.
	// Providers can add their own functions, this list is used as a
	// reference to provide a better user experience in giving prompts
	// to make sure the user is aware when a function is not a core function,
	// so they can check that a provider is correctly configured.
	CoreSubstitutionFunctions = []SubstitutionFunctionName{
		SubstitutionFunctionFromJSON,
		SubstitutionFunctionFromJSON_G,
		SubstitutionFunctionJSONDecode,
		SubstitutionFunctionLen,
		SubstitutionFunctionSubstr,
		SubstitutionFunctionSubstr_G,
		SubstitutionFunctionReplace,
		SubstitutionFunctionReplace_G,
		SubstitutionFunctionTrim,
		SubstitutionFunctionTrimPrefix,
		SubstitutionFunctionTrimPrefix_G,
		SubstitutionFunctionTrimSuffix,
		SubstitutionFunctionTrimSuffix_G,
		SubstitutionFunctionSplit,
		SubstitutionFunctionSplit_G,
		SubstitutionFunctionJoin,
		SubstitutionFunctionIndex,
		SubstitutionFunctionLastIndex,
		SubstitutionFunctionToUpper,
		SubstitutionFunctionToLower,
		SubstitutionFunctionHasPrefix,
		SubstitutionFunctionHasPrefix_G,
		SubstitutionFunctionHasSuffix,
		SubstitutionFunctionHasSuffix_G,
		SubstitutionFunctionContains,
		SubstitutionFunctionContains_G,
		SubstitutionFunctionList,
		SubstitutionFunctionObject,
		SubstitutionFunctionKeys,
		SubstitutionFunctionVals,
		SubstitutionFunctionMap,
		SubstitutionFunctionFilter,
		SubstitutionFunctionReduce,
		SubstitutionFunctionSort,
		SubstitutionFunctionFlatMap,
		SubstitutionFunctionCompose,
		SubstitutionFunctionPipe,
		SubstitutionFunctionGetAttr,
		SubstitutionFunctionGetElem,
		SubstitutionFunctionLink,
		SubstitutionFunctionAnd,
		SubstitutionFunctionOr,
		SubstitutionFunctionNot,
		SubstitutionFunctionEq,
		SubstitutionFunctionGT,
		SubstitutionFunctionGE,
		SubstitutionFunctionLT,
		SubstitutionFunctionLE,
		SubstitutionFunctionCWD,
		SubstitutionFunctionDateTime,
	}
)
