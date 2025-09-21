package container

// MaxBlueprintDepth is the maximum depth allowed for a tree of blueprints
// referenced by the use of the `include` feature in a blueprint.
// This means that if you have a dependency tree:
//
//	blueprint1
//	├── blueprint2
//	│	 ├── blueprint3
//	│	 │   └── blueprint4
//
// The depth of the the tree would be 4, therefore causing an error
// if MaxBlueprintDepth is set to 3.
//
// This will also help with breaking out of cyclic blueprint inclusions.
// For example, if we have the following cycle and max depth is 3:
//
//	├── blueprint1
//	│   ├── blueprint2
//	│   │   ├── blueprint3
//	│   │   │   └── blueprint1
//
// The iterations will be:
// 1. process blueprint1
// 2. process blueprint2 included in blueprint1
// 3. process blueprint3 included in blueprint2
// 4. Fail to due max depth reached.
const MaxBlueprintDepth = 5
