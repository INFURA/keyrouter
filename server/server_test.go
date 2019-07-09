package server_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/INFURA/keyrouter/consistent"
	"github.com/INFURA/keyrouter/server"
)

func TestServer_ServiceQueryHandler(t *testing.T) {

	srv := server.New()
	// add some entries
	{
		_, _, err := srv.PopulateService("foo", consistent.Members{"a", "b", "c"})
		require.NoError(t, err)
	}

	{
		_, _, err := srv.PopulateService("bar", consistent.Members{"1", "2", "3"})
		require.NoError(t, err)
	}

	handler := srv.ServiceQueryHandler()

	// get 3 entries for foo
	{
		data := url.Values{
			"key": {"k"},
			"min": {"3"},
			"max": {"3"},
		}
		request, err := http.NewRequest("POST", "foo", strings.NewReader(data.Encode()))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, request)

		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	}

	// remove an entry from foo
	{
		_, _, err := srv.PopulateService("foo", consistent.Members{"a", "b"})
		require.NoError(t, err)
	}

	// try again to get 3 entries, which should now fail
	{
		data := url.Values{
			"key": {"k"},
			"min": {"3"},
			"max": {"3"},
		}
		request, err := http.NewRequest("POST", "foo", strings.NewReader(data.Encode()))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, request)

		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	// try to get between 2 and 3 entries, which should work
	{
		data := url.Values{
			"key": {"k"},
			"min": {"2"},
			"max": {"3"},
		}
		request, err := http.NewRequest("POST", "foo", strings.NewReader(data.Encode()))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, request)

		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	}

	// JSON should work as well
	{
		request, err := http.NewRequest("POST", "foo", bytes.NewBuffer([]byte(`{"key":"k","min":1,"max":1}`)))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, request)

		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	}

	// try to get a service we've never seen, which should 404
	{
		data := url.Values{
			"key": {"k"},
			"min": {"1"},
			"max": {"1"},
		}
		request, err := http.NewRequest("POST", "unknown", strings.NewReader(data.Encode()))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, request)

		require.Equal(t, http.StatusNotFound, rr.Code, rr.Body.String())
	}

	// missing arguments should fail
	{
		data := url.Values{}
		request, err := http.NewRequest("POST", "foo", strings.NewReader(data.Encode()))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, request)

		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	// min > max should also fail
	{
		data := url.Values{
			"key": {"k"},
			"min": {"3"},
			"max": {"1"},
		}
		request, err := http.NewRequest("POST", "foo", strings.NewReader(data.Encode()))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, request)

		require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestServer_AllServicesHandler(t *testing.T) {
	srv := server.New()
	// add some entries
	{
		_, _, err := srv.PopulateService("foo", consistent.Members{"a", "b", "c"})
		require.NoError(t, err)
	}

	{
		_, _, err := srv.PopulateService("bar", consistent.Members{"1", "2", "3"})
		require.NoError(t, err)
	}

	handler := srv.AllServicesHandler()

	// do a GET on the handler
	{
		request, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, request)

		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	}
}

func BenchmarkServer_ServiceQueryHandler(b *testing.B) {
	srv := server.New()

	services := 10
	nodes := 100

	for service := 0; service < services; service++ {
		members := make(consistent.Members, 50)
		for node := 0; node < nodes; node++ {
			members = append(members, consistent.Member(fmt.Sprintf("node-%d", node)))
		}

		_, _, err := srv.PopulateService(fmt.Sprintf("service-%d", service), members)
		require.NoError(b, err)
	}
	handler := srv.ServiceQueryHandler()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		data := url.Values{
			"key": {fmt.Sprintf("key-%d", n)},
			"min": {"3"},
			"max": {"1"},
		}
		request, _ := http.NewRequest("POST", fmt.Sprintf("service-%d", n%services), strings.NewReader(data.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, request)
	}
}
