package eino

type einoRequest struct {
	operationName string
	input         map[string]any
	output        map[string]any
}

type einoLLMRequest struct {
	operationName    string
	modelName        string
	encodingFormats  []string
	frequencyPenalty float64
	presencePenalty  float64
	maxTokens        int64
	usageInputTokens int64
	stopSequences    []string
	temperature      float64
	topK             float64
	topP             float64
	serverAddress    string
	seed             int64
}
type einoLLMResponse struct {
	responseFinishReasons []string
	responseModel         string
	usageOutputTokens     int64
	responseID            string
}
