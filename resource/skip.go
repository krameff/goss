package resource

import "time"

// SkipResourceResults emits skip results for every validated property on a resource.
func SkipResourceResults(res Resource, reason string) []TestResult {
	startTime := time.Now()
	validateErr := ValidateError(reason)

	if rr, ok := res.(ResourceRead); ok {
		return []TestResult{
			{
				Result:       SKIP,
				Skipped:      true,
				ResourceType: res.TypeName(),
				ResourceId:   rr.ID(),
				Title:        rr.GetTitle(),
				Meta:         rr.GetMeta(),
				Property:     "depends-on",
				Err:          &validateErr,
				StartTime:    startTime,
				EndTime:      startTime,
			},
		}
	}

	return []TestResult{
		{
			Result:       SKIP,
			Skipped:      true,
			ResourceType: res.TypeName(),
			ResourceId:   res.TypeKey(),
			Property:     "depends-on",
			Err:          &validateErr,
			StartTime:    startTime,
			EndTime:      startTime,
		},
	}
}
