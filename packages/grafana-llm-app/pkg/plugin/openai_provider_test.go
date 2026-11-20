package plugin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenAIProvider_NilAPIPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":"hi"}}]}`))
	}))
	defer srv.Close()

	// APIPath is nil when settings are not built by loadSettings.
	var p LLMProvider
	var err error
	require.NotPanics(t, func() {
		p, err = NewOpenAIProvider(OpenAISettings{URL: srv.URL}, nil)
	})
	require.NoError(t, err)

	_, err = p.ChatCompletion(context.Background(), ChatCompletionRequest{Model: ModelBase})
	require.NoError(t, err)
	assert.Equal(t, "/v1/chat/completions", gotPath)
}
