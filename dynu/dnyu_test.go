package dynu_test

import (
	"net"
	"net/http"
	"net/http/httputil"
	"testing"

	"github.com/justenwalker/ddns/dynu"
)

type testRequester struct {
	t    *testing.T
	resp *http.Response
}

func (r testRequester) Do(req *http.Request) (*http.Response, error) {
	reqbody, _ := httputil.DumpRequest(req, false)
	r.t.Log("<< REQUEST\n", string(reqbody))
	return r.resp, nil
}

func TestUpdateIP(t *testing.T) {
	client := dynu.New("foo", "bar",
		dynu.HTTPClient(testRequester{t: t, resp: &http.Response{}}),
		dynu.Hostnames([]string{"dionysus.myddns.rocks"}),
	)
	err := client.UpdateIP([]net.IP{
		net.IP([]byte{14, 14, 22, 149}),
	})
	if err != nil {
		t.Fatal(err)
	}
}
