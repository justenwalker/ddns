package dynu // import "github.com/justenwalker/ddns/dynu"

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const apiEndpoint = "https://api.dynu.com"
const updatePath = "/nic/update"

// Logger for printing debug logs from this package
type Logger interface {
	Log(format string, v ...interface{})
}

// HTTPRequester makes http requests and returns responses
// *http.Client implicitly implements Requester and can be provided whever this interface is requested.
type HTTPRequester interface {
	Do(req *http.Request) (*http.Response, error)
}

// Option sets client options
type Option func(*Client)

// Client for communicating with the IP Update API at dynu.com
type Client struct {
	logger     Logger
	ipv6       bool
	ipv4       bool
	httpClient HTTPRequester
	endpoint   string
	username   string
	password   string
	location   string
	hostnames  []string
}

// Log enables client logging using the given Logger
func Log(l Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// IPv6 enables/disables setting the IPv6 address
func IPv6(enabled bool) Option {
	return func(c *Client) {
		c.ipv6 = enabled
	}
}

// IPv4 enables/disables setting the IPv4 address
func IPv4(enabled bool) Option {
	return func(c *Client) {
		c.ipv4 = enabled
	}
}

// Hostnames whose IP address requires update.
// Clears the 'Location' option when used.
func Hostnames(hostnames []string) Option {
	return func(c *Client) {
		c.hostnames = hostnames
		c.location = ""
	}
}

// Location to update IP address for a collection of hostnames including those created using subdomains.
// The Hostnames option is cleared when this option is provided
func Location(location string) Option {
	return func(c *Client) {
		c.location = location
		c.hostnames = nil
	}
}

// Endpoint sets the API Endpoint of the dynu.com API
// The default should normally be fine
func Endpoint(endpoint string) Option {
	return func(c *Client) {
		c.endpoint = endpoint
	}
}

// HTTPClient sets a custom HTTP client to use for all of the API calls
// the default uses http.DefaultClient
func HTTPClient(hc HTTPRequester) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// New constructs a dnyu.com API client
func New(username string, password string, options ...Option) *Client {
	client := &Client{
		username:   username,
		password:   password,
		endpoint:   apiEndpoint,
		httpClient: http.DefaultClient,
		ipv6:       false,
		ipv4:       true,
	}
	for _, opt := range options {
		opt(client)
	}
	return client
}

func hashPassword(password string) string {
	bs := sha256.Sum256([]byte(password))
	return hex.EncodeToString(bs[:])
}

// DoUpdateIP executes the UpdateIP request and returns the response
func (c *Client) DoUpdateIP(ips []net.IP) (*Response, error) {
	// URL Format:
	// https://api.dynu.com/nic/update?hostname=[HOSTNAME]&myip=[IP ADDRESS]&myipv6=[IPv6 ADDRESS]&password=[PASSWORD or MD5(PASSWORD) or SHA256(PASSWORD)]
	// https://api.dynu.com/nic/update?username=[USERNAME]&myip=[IP ADDRESS]&myipv6=[IPv6 ADDRESS]&password=[PASSWORD or MD5(PASSWORD) or SHA256(PASSWORD)]
	uri, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, err
	}
	q := make(url.Values)
	q.Set("password", hashPassword(c.password))
	if len(c.hostnames) > 0 {
		q.Set("hostname", strings.Join(c.hostnames, ","))
	} else {
		q.Set("username", c.username)
		if c.location != "" {
			q.Set("location", c.location)
		}
	}
	q.Set("myip", "no")
	q.Set("myipv6", "no")
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil { // IPv4
			if c.ipv4 {
				q.Set("myip", ipv4.String())
			}
		} else { // IPv6
			if c.ipv6 {
				q.Set("myipv6", ip.String())
			}
		}
	}
	uri.Path = updatePath
	uri.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	rs, err := ReadResponse(bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	return rs, nil
}

// UpdateIP updates the ip address of the dnyu address
func (c *Client) UpdateIP(ips []net.IP) error {
	rs, err := c.DoUpdateIP(ips)
	if err != nil {
		return err
	}
	return rs.ToError()
}
