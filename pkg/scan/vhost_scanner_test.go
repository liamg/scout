package scan

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVHOSTScanner(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		switch r.Host {
		case "site.eg", "admin.site.eg":
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

	parts := strings.Split(parsed.Host, ":")

	host := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Error(err)
	}

	scanner := NewVHOSTScanner(&VHOSTOptions{
		BaseDomain:  "site.eg",
		IP:          host,
		Port:        port,
		Parallelism: 1,
	})

	results, err := scanner.Scan()
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, len(results), 1)
	assert.Equal(t, results[0], "admin.site.eg")

}
