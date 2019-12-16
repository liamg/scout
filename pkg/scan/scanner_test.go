package scan

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanner(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login.php":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	scanner := NewScanner(&Options{
		TargetURL:   *parsed,
		Parallelism: 100,
	})

	results, err := scanner.Scan()
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, len(results), 1)
	assert.Equal(t, results[0].String(), server.URL+"/login.php")

}
