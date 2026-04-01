package provider

func BuildBaseOptions(temperature *float64, maxTokens *int, apiKey *string) StreamOptions {
	return StreamOptions{
		Temperature: temperature,
		MaxTokens:   maxTokens,
		APIKey:      apiKey,
	}
}

func AdjustMaxTokensForThinking(opts StreamOptions, level ThinkingLevel, budgets *ThinkingBudgets) StreamOptions {
	if level == "" || opts.MaxTokens == nil || budgets == nil {
		return opts
	}

	var budget *int
	switch level {
	case ThinkingLevelMinimal:
		budget = budgets.Minimal
	case ThinkingLevelLow:
		budget = budgets.Low
	case ThinkingLevelMedium:
		budget = budgets.Medium
	case ThinkingLevelHigh, ThinkingLevelXHigh:
		budget = budgets.High
	}

	if budget == nil {
		return opts
	}

	adjusted := *opts.MaxTokens - *budget
	opts.MaxTokens = &adjusted
	return opts
}
