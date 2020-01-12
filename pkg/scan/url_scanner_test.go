package scan

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/liamg/scout/pkg/wordlist"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLScanner(t *testing.T) {

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

	scanner := NewURLScanner(&URLOptions{
		TargetURL:   *parsed,
		Parallelism: 2,
		Wordlist:    wordlist.FromReader(bytes.NewReader([]byte("login.php\nsomething.php"))),
	})

	results, err := scanner.Scan()
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 1, len(results))
	assert.Equal(t, results[0].String(), server.URL+"/login.php")

}

func TestURLScannerWithRedirects(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/very-secret-file.php":
			w.WriteHeader(http.StatusOK)
		case "/login.php":
			http.Redirect(w, r, "/very-secret-file.php", http.StatusFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	scanner := NewURLScanner(&URLOptions{
		TargetURL:           *parsed,
		Parallelism:         1,
		PositiveStatusCodes: []int{http.StatusOK},
		Wordlist:            wordlist.FromReader(bytes.NewReader([]byte("login.php"))),
	})

	results, err := scanner.Scan()
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 1, len(results))
	assert.Equal(t, results[0].String(), server.URL+"/very-secret-file.php")

}

func TestURLScannerWithBackupFile(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login.php":
			w.WriteHeader(http.StatusOK)
		case "/login.php~":
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

	scanner := NewURLScanner(&URLOptions{
		TargetURL:   *parsed,
		Parallelism: 1,
		Wordlist:    wordlist.FromReader(bytes.NewReader([]byte("login.php"))),
	})

	results, err := scanner.Scan()
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 2, len(results))
	assert.Equal(t, results[0].String(), server.URL+"/login.php")
	assert.Equal(t, results[1].String(), server.URL+"/login.php~")

}
