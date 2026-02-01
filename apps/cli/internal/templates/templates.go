package templates

// Template represents a blueprint project template maintained by the Bluelink team.
type Template struct {
	Key         string
	Label       string
	Description string
}

// GetTemplates returns all available blueprint project templates.
func GetTemplates() []Template {
	return []Template{
		{
			Key:   "scaffold",
			Label: "Scaffold",
			Description: "A scaffold project that generates essential files " +
				"with placeholders.",
		},
		{
			Key:   "aws-simple-api",
			Label: "AWS Simple API",
			Description: "A simple API project using AWS API Gateway and " +
				"Lambda functions for a RESTful API.",
		},
	}
}
