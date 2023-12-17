package locale

// EN locale validation message.
var EN = map[string]any{
	"required":        "The %s field is required.",
	"string":          "The %s must be a string.",
	"alpha":           "The %s may only contain letters.",
	"numeric":         "The %s must be a number.",
	"alpha_numeric":   "The %s may only contain letters and numbers.",
	"int":             "The %s must be an integer.",
	"uint":            "The %s must be a positive integer.",
	"float":           "The %s must be a float.",
	"email":           "The %s must be a valid email address.",
	"phone":           "The %s must be a valid phone number.",
	"phone_with_code": "The %s must be a valid phone number with country code.",
	"username":        "The %s must be a valid email address or phone number or phone number with country code.",
	"match":           "The %s does not matched.",
	"same":            "The %s and %s must match.",
	"unique":          "The %s has already been taken.",
	"bool":            "The %s field must be true.",
	"file":            "The %s must be a file.",
	"file_type":       "The %s must be a file of type: %s.",
	"image":           "The %s must be an image.",
	"image_type":      "The %s must be an image of type: %s.",
	"mimes":           "The %s must be a file of type: %s.",
	"gh_card":         "The %s must be a valid Ghana Card.",
	"gh_gps":          "The %s must be a valid Ghana digital address.",
	"gt": map[string]string{
		"numeric": "The %s must be greater than %s.",
		"file":    "The %s must be greater than %s megabytes.",
		"string":  "The %s must be greater than %s characters.",
		"slice":   "The %s must have more than %s items.",
	},
	"gte": map[string]string{
		"numeric": "The %s must be greater than or equal to %s.",
		"file":    "The %s must be greater than or equal to %s megabytes.",
		"string":  "The %s must be greater than or equal to %s characters.",
		"slice":   "The %s must have %s items or more.",
	},
	"lt": map[string]string{
		"numeric": "The %s must be less than %s.",
		"file":    "The %s must be less than %s megabytes.",
		"string":  "The %s must be less than %s characters.",
		"slice":   "The %s must have less than %s items.",
	},
	"lte": map[string]string{
		"numeric": "The %s must be less than or equal to %s.",
		"file":    "The %s must be less than or equal to %s megabytes.",
		"string":  "The %s must be less than or equal to %s characters.",
		"slice":   "The %s must not have more than %s items.",
	},
	"min": map[string]string{
		"numeric": "The %s must be at least %s",
		"file":    "The %s must be at least %s megabytes.",
		"string":  "The %s must be at least %s characters.",
		"slice":   "The %s must have at least %s items.",
	},
	"max": map[string]string{
		"numeric": "The %s must not be greater than %s.",
		"file":    "The %s must not be greater than %s megabytes.",
		"string":  "The %s must not be greater than %s characters.",
		"slice":   "The %s must not have more than %s items.",
	},
	"equal": map[string]string{
		"numeric": "The %s must be equal to %s.",
		"file":    "The %s must be equal to %s megabytes.",
		"string":  "The %s must be equal to %s characters.",
		"slice":   "The %s must be equal to %s items.",
	},
	"between": map[string]string{
		"numeric": "The %s must be between %s and %s.",
		"file":    "The %s must be between %s and %s megabytes.",
		"string":  "The %s must be between %s and %s characters.",
		"slice":   "The %s must be between %s and %s items.",
	},
	"from": map[string]string{
		"numeric": "The %s must be from %s to %s.",
		"file":    "The %s must be from %s to %s megabytes.",
		"string":  "The %s must be from %s to %s characters.",
		"slice":   "The %s must be from %s to %s items.",
	},
	"size": map[string]string{
		"numeric": "The %s must be %s.",
		"file_kb": "The %s must be %s kilobytes.",
		"file_mb": "The %s must be %s megabytes.",
		"file_gb": "The %s must be %s gigabytes.",
		"string":  "The %s must be %s characters.",
		"slice":   "The %s must contain %s items.",
	},
	"date": map[string]string{
		"rfc3339":  "The %s does not match the format: 2006-01-02T00:00:00Z",
		"datetime": "The %s does not match the format: 2006-01-02 15:04:05",
		"dateonly": "The %s does not match the format: 2006-01-02",
	},
}