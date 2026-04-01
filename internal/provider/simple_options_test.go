package provider

import "testing"

func TestBuildBaseOptionsDefaults(t *testing.T) {
	opts := BuildBaseOptions(nil, nil, nil)

	if opts.Temperature != nil {
		t.Error("expected nil Temperature")
	}
	if opts.MaxTokens != nil {
		t.Error("expected nil MaxTokens")
	}
	if opts.APIKey != nil {
		t.Error("expected nil APIKey")
	}
}

func TestBuildBaseOptionsWithValues(t *testing.T) {
	temp := 0.7
	tokens := 4096
	key := "sk-test"

	opts := BuildBaseOptions(&temp, &tokens, &key)

	if opts.Temperature == nil || *opts.Temperature != 0.7 {
		t.Errorf("expected Temperature 0.7, got %v", opts.Temperature)
	}
	if opts.MaxTokens == nil || *opts.MaxTokens != 4096 {
		t.Errorf("expected MaxTokens 4096, got %v", opts.MaxTokens)
	}
	if opts.APIKey == nil || *opts.APIKey != "sk-test" {
		t.Errorf("expected APIKey sk-test, got %v", opts.APIKey)
	}
}

func TestBuildBaseOptionsPartial(t *testing.T) {
	tokens := 2048
	opts := BuildBaseOptions(nil, &tokens, nil)

	if opts.Temperature != nil {
		t.Error("expected nil Temperature")
	}
	if opts.MaxTokens == nil || *opts.MaxTokens != 2048 {
		t.Errorf("expected MaxTokens 2048, got %v", opts.MaxTokens)
	}
	if opts.APIKey != nil {
		t.Error("expected nil APIKey")
	}
}

func TestAdjustMaxTokensForThinkingNoThinking(t *testing.T) {
	opts := BuildBaseOptions(nil, nil, nil)
	result := AdjustMaxTokensForThinking(opts, ThinkingLevel(""), nil)

	if result.MaxTokens != nil {
		t.Error("expected nil MaxTokens when no thinking level")
	}
}

func TestAdjustMaxTokensForThinkingWithMaxTokens(t *testing.T) {
	tokens := 16384
	opts := BuildBaseOptions(nil, &tokens, nil)
	budgets := ThinkingBudgets{
		High:   intPtr(10000),
		Medium: intPtr(5000),
		Low:    intPtr(2000),
	}

	result := AdjustMaxTokensForThinking(opts, ThinkingLevelHigh, &budgets)

	if result.MaxTokens == nil {
		t.Fatal("expected non-nil MaxTokens")
	}
	if *result.MaxTokens != 6384 {
		t.Errorf("expected MaxTokens 6384 (16384 - 10000), got %d", *result.MaxTokens)
	}
}

func TestAdjustMaxTokensForThinkingSubtractsBudget(t *testing.T) {
	tokens := 16384
	opts := BuildBaseOptions(nil, &tokens, nil)
	budgets := ThinkingBudgets{
		High:   intPtr(10000),
		Medium: intPtr(5000),
		Low:    intPtr(2000),
	}

	result := AdjustMaxTokensForThinking(opts, ThinkingLevelMedium, &budgets)

	if result.MaxTokens == nil {
		t.Fatal("expected non-nil MaxTokens")
	}
	if *result.MaxTokens != 11384 {
		t.Errorf("expected MaxTokens 11384 (16384 - 5000), got %d", *result.MaxTokens)
	}
}

func TestAdjustMaxTokensForThinkingNoMaxTokens(t *testing.T) {
	opts := BuildBaseOptions(nil, nil, nil)
	budgets := ThinkingBudgets{High: intPtr(10000)}

	result := AdjustMaxTokensForThinking(opts, ThinkingLevelHigh, &budgets)

	if result.MaxTokens != nil {
		t.Error("expected nil MaxTokens when input has no MaxTokens")
	}
}

func TestAdjustMaxTokensForThinkingNoBudgets(t *testing.T) {
	tokens := 8192
	opts := BuildBaseOptions(nil, &tokens, nil)

	result := AdjustMaxTokensForThinking(opts, ThinkingLevelHigh, nil)

	if result.MaxTokens == nil {
		t.Fatal("expected non-nil MaxTokens")
	}
	if *result.MaxTokens != 8192 {
		t.Errorf("expected MaxTokens 8192 (unchanged), got %d", *result.MaxTokens)
	}
}

func TestAdjustMaxTokensForThinkingLowLevel(t *testing.T) {
	tokens := 8192
	opts := BuildBaseOptions(nil, &tokens, nil)
	budgets := ThinkingBudgets{Low: intPtr(2000)}

	result := AdjustMaxTokensForThinking(opts, ThinkingLevelLow, &budgets)

	if *result.MaxTokens != 6192 {
		t.Errorf("expected MaxTokens 6192, got %d", *result.MaxTokens)
	}
}

func TestAdjustMaxTokensForThinkingMinimalLevel(t *testing.T) {
	tokens := 8192
	opts := BuildBaseOptions(nil, &tokens, nil)
	budgets := ThinkingBudgets{Minimal: intPtr(1024)}

	result := AdjustMaxTokensForThinking(opts, ThinkingLevelMinimal, &budgets)

	if *result.MaxTokens != 7168 {
		t.Errorf("expected MaxTokens 7168, got %d", *result.MaxTokens)
	}
}

func TestAdjustMaxTokensForThinkingMissingSpecificBudget(t *testing.T) {
	tokens := 8192
	opts := BuildBaseOptions(nil, &tokens, nil)
	budgets := ThinkingBudgets{Low: intPtr(2000)}

	result := AdjustMaxTokensForThinking(opts, ThinkingLevelHigh, &budgets)

	if *result.MaxTokens != 8192 {
		t.Errorf("expected MaxTokens 8192 (no high budget defined), got %d", *result.MaxTokens)
	}
}

func TestAdjustMaxTokensForThinkingXHighLevel(t *testing.T) {
	tokens := 8192
	opts := BuildBaseOptions(nil, &tokens, nil)
	budgets := ThinkingBudgets{High: intPtr(6000)}

	result := AdjustMaxTokensForThinking(opts, ThinkingLevelXHigh, &budgets)

	if *result.MaxTokens != 2192 {
		t.Errorf("expected MaxTokens 2192 (xhigh uses high budget), got %d", *result.MaxTokens)
	}
}
