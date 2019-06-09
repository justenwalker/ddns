package dynu

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

// Error encapsualtes response code errors and their mapping to the request in a multi-request response
type Error struct {
	Request int
	Code    ResponseCode
	Detail  string
}

func (e Error) Error() string {
	if e.Code != "" {
		if e.Detail != "" {
			return fmt.Sprintf("%s: %s", e.Code, e.Detail)
		}
		return fmt.Sprintf("%s", e.Code)
	}
	return "unknown error"
}

// Temporary returns true if the error is temporary and may succeed after a retry
func (e Error) Temporary() bool {
	switch e.Code {
	case RespDNS, Resp911, RespServerError:
		return true
	}
	return false
}

// ResponseErrors implements the error interface for multi-request responses
type ResponseErrors []Error

// ResponseErrors returns the response error string
func (rs ResponseErrors) Error() string {
	buf := &bytes.Buffer{}
	buf.WriteString(fmt.Sprintf("dynu: response return %d error(s):", len(rs)))
	for _, r := range rs {
		buf.WriteString(fmt.Sprintf("\n\t* [%d] %v", r.Request, r.Error()))
	}
	return buf.String()
}

// ResponseCode responses from the IP Update API
type ResponseCode string

// Response contains the response code for each request
type Response struct {
	Codes  []ResponseCode
	Detail []string
}

// ToError returns the response errors, or nil if there were no errors
func (rs Response) ToError() error {
	var errs ResponseErrors
	for i, r := range rs.Codes {
		if r.IsError() {
			errs = append(errs, Error{
				Request: i,
				Code:    r,
				Detail:  rs.Detail[i],
			})
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// ReadResponse reads the response code from the http response body
func ReadResponse(r io.Reader) (*Response, error) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var response Response
	codes := strings.Split(string(body), "\n\r")
	for _, code := range codes {
		sp := strings.SplitN(code, " ", 2)
		rc := ResponseCode(strings.TrimSpace(strings.ToLower(sp[0])))
		detail := ""
		if len(sp) > 1 {
			detail = sp[1]
		}
		response.Codes = append(response.Codes, rc)
		response.Detail = append(response.Detail, detail)
	}
	return &response, nil
}

// IsError returns true if the response code is an error
func (rc ResponseCode) IsError() bool {
	switch rc {
	case RespGood, RespNoChange:
		return false
	}
	return true
}

const (
	// RespUnknown This response code is returned if an invalid 'request' is made to the API server. This 'response code' could be generated as a result of badly formatted parameters as well so parameters must be checked for validity by the client before they are passed along with the 'request'.
	RespUnknown = ResponseCode("unknown")

	// RespGood This response code means that the action has been processed successfully. Further details of the action may be included along with this 'response code'.
	RespGood = ResponseCode("good")

	// RespBadAuth This response code is returned in case of a failed authentication for the 'request'. Please note that sending across an invalid parameter such as an unknown domain name can also result in this 'response code'. The client must advise the user to check all parameters including authentication parameters to resolve this problem.
	RespBadAuth = ResponseCode("badauth")

	// RespServerError This response code is returned in cases where an error was encountered on the server side. The client may send across the request again to have the 'request' processed successfully.
	RespServerError = ResponseCode("servererror")

	// RespNoChange This response code is returned in cases where IP address was found to be unchanged on the server side.
	RespNoChange = ResponseCode("nochg")

	// RespNotFQDN This response code is returned in cases where the hostname is not a valid fully qualified hostname.
	RespNotFQDN = ResponseCode("notfqdn")

	// RespNumHost This response code is returned in cases where too many hostnames(more than 20) are specified for the update process.
	RespNumHost = ResponseCode("numhost")

	// RespAbuse This response code is returned in cases where update process has failed due to abusive behaviour.
	RespAbuse = ResponseCode("abuse")

	// RespNohost This response code is returned in cases where hostname/username is not found in the system.
	RespNohost = ResponseCode("nohost")

	// Resp911 This response code is returned in cases where the update is temporarily halted due to scheduled maintenance. Client must respond by suspending update process for 10 minutes upon receiving this response code.
	Resp911 = ResponseCode("911")

	// RespDNS This response code is returned in cases where there was an error on the server side. The client must respond by retrying the update process.
	RespDNS = ResponseCode("dnserr")

	// RespNotDonator This response code is returned to indicate that this functionality is only available to members.
	RespNotDonator = ResponseCode("!donator")
)
