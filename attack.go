package cmrdr

import (
	"fmt"
	"time"

	curl "github.com/andelf/go-curl"
	"github.com/pkg/errors"
	v "gopkg.in/go-playground/validator.v9"
)

// HTTP responses
const (
	httpOK           = 200
	httpUnauthorized = 401
	httpForbidden    = 403
	httpNotFound     = 404
)

// CURL RTSP request types
const (
	rtspDescribe = 2
	rtspSetup    = 4
)

// HACK: See https://stackoverflow.com/questions/3572397/lib-curl-in-c-disable-printing
func doNotWrite([]uint8, interface{}) bool {
	return true
}

func routeAttack(c Curler, stream Stream, route string, timeout time.Duration, enableLogs bool) bool {
	attackURL := fmt.Sprintf(
		"rtsp://%s:%s@%s:%d/%s",
		stream.Username,
		stream.Password,
		stream.Address,
		stream.Port,
		route,
	)

	if enableLogs {
		// Debug logs when logs are enabled
		c.Setopt(curl.OPT_VERBOSE, 1)
	} else {
		// Do not write sdp in stdout
		c.Setopt(curl.OPT_WRITEFUNCTION, doNotWrite)
	}

	// Do not use signals (would break multithreading)
	c.Setopt(curl.OPT_NOSIGNAL, 1)
	// Do not send a body in the describe request
	c.Setopt(curl.OPT_NOBODY, 1)
	// Send a request to the URL of the stream we want to attack
	c.Setopt(curl.OPT_URL, attackURL)
	// Set the RTSP STREAM URI as the stream URL
	c.Setopt(curl.OPT_RTSP_STREAM_URI, attackURL)
	// 2 is CURL_RTSPREQ_DESCRIBE
	c.Setopt(curl.OPT_RTSP_REQUEST, rtspDescribe)
	// Set custom timeout
	c.Setopt(curl.OPT_TIMEOUT_MS, int(timeout/time.Millisecond))

	// Perform the request
	err := c.Perform()
	if err != nil {
		fmt.Printf("\nERROR: curl timeout on stream '%s' reached after %s.\nconsider increasing the timeout (-T, --timeout parameter) to at least 5000ms if scanning an unstable network.\n", stream.Address, timeout.String())
		return false
	}

	// Get return code for the request
	rc, err := c.Getinfo(curl.INFO_RESPONSE_CODE)
	if err != nil {
		return false
	}

	// If it's a 401 or 403, it means that the credentials are wrong but the route might be okay
	// If it's a 200, the stream is accessed successfully
	if rc == httpOK || rc == httpUnauthorized || rc == httpForbidden {
		return true
	}
	return false
}

func credAttack(c Curler, stream Stream, username string, password string, timeout time.Duration, enableLogs bool) bool {
	attackURL := fmt.Sprintf(
		"rtsp://%s:%s@%s:%d/%s",
		username,
		password,
		stream.Address,
		stream.Port,
		stream.Route,
	)

	if enableLogs {
		// Debug logs when logs are enabled
		c.Setopt(curl.OPT_VERBOSE, 1)
	} else {
		// Do not write sdp in stdout
		c.Setopt(curl.OPT_WRITEFUNCTION, doNotWrite)
	}

	// Do not use signals (would break multithreading)
	c.Setopt(curl.OPT_NOSIGNAL, 1)
	// Do not send a body in the describe request
	c.Setopt(curl.OPT_NOBODY, 1)
	// Send a request to the URL of the stream we want to attack
	c.Setopt(curl.OPT_URL, attackURL)
	// Set the RTSP STREAM URI as the stream URL
	c.Setopt(curl.OPT_RTSP_STREAM_URI, attackURL)
	// 2 is CURL_RTSPREQ_DESCRIBE
	c.Setopt(curl.OPT_RTSP_REQUEST, 2)
	// Set custom timeout
	c.Setopt(curl.OPT_TIMEOUT_MS, int(timeout/time.Millisecond))

	// Perform the request
	err := c.Perform()
	if err != nil {
		fmt.Printf("\nERROR: curl timeout on stream '%s' reached after %s.\nconsider increasing the timeout (-T, --timeout parameter) to at least 5000ms if scanning an unstable network.\n", stream.Address, timeout.String())
		return false
	}

	// Get return code for the request
	rc, err := c.Getinfo(curl.INFO_RESPONSE_CODE)
	if err != nil {
		return false
	}

	// If it's a 404, it means that the route is incorrect but the credentials might be okay
	// If it's a 200, the stream is accessed successfully
	if rc == httpOK || rc == httpNotFound {
		return true
	}
	return false
}

func validateStream(c Curler, stream Stream, timeout time.Duration, enableLogs bool) bool {
	attackURL := fmt.Sprintf(
		"rtsp://%s:%s@%s:%d/%s",
		stream.Username,
		stream.Password,
		stream.Address,
		stream.Port,
		stream.Route,
	)

	if enableLogs {
		// Debug logs when logs are enabled
		c.Setopt(curl.OPT_VERBOSE, 1)
	} else {
		// Do not write sdp in stdout
		c.Setopt(curl.OPT_WRITEFUNCTION, doNotWrite)
	}

	// Do not use signals (would break multithreading)
	c.Setopt(curl.OPT_NOSIGNAL, 1)
	// Do not send a body in the describe request
	c.Setopt(curl.OPT_NOBODY, 1)
	// Send a request to the URL of the stream we want to attack
	c.Setopt(curl.OPT_URL, attackURL)
	// Set the RTSP STREAM URI as the stream URL
	c.Setopt(curl.OPT_RTSP_STREAM_URI, attackURL)
	// 2 is CURL_RTSPREQ_SETUP
	c.Setopt(curl.OPT_RTSP_REQUEST, rtspSetup)
	// Set custom timeout
	c.Setopt(curl.OPT_TIMEOUT_MS, int(timeout/time.Millisecond))

	c.Setopt(curl.OPT_RTSP_TRANSPORT, "RTP/AVP;unicast;client_port=33332-33333")

	// Perform the request
	err := c.Perform()
	if err != nil {
		fmt.Printf("\nERROR: curl timeout on stream '%s' reached after %s.\nconsider increasing the timeout (-T, --timeout parameter) to at least 5000ms if scanning an unstable network.\n", stream.Address, timeout.String())
		return false
	}

	// Get return code for the request
	rc, err := c.Getinfo(curl.INFO_RESPONSE_CODE)
	if err != nil {
		return false
	}

	// If it's a 200, the stream is accessed successfully
	if rc == httpOK {
		return true
	}
	return false
}

// ValidateStreams tries to setup the stream to validate whether or not it is available
func ValidateStreams(c Curler, targets []Stream, timeout time.Duration, log bool) ([]Stream, error) {
	for idx, target := range targets {
		targets[idx].Available = validateStream(c, target, timeout, log)
	}

	return targets, nil
}

func attackCameraCredentials(c Curler, target Stream, credentials Credentials, resultsChan chan<- Stream, timeout time.Duration, log bool) {
	for _, username := range credentials.Usernames {
		for _, password := range credentials.Passwords {
			ok := credAttack(c.Duphandle(), target, username, password, timeout, log)
			if ok {
				target.CredentialsFound = true
				target.Username = username
				target.Password = password
				resultsChan <- target
				return
			}
		}
	}
	target.CredentialsFound = false
	resultsChan <- target
}

func attackCameraRoute(c Curler, target Stream, routes Routes, resultsChan chan<- Stream, timeout time.Duration, log bool) {
	for _, route := range routes {
		ok := routeAttack(c.Duphandle(), target, route, timeout, log)
		if ok {
			target.RouteFound = true
			target.Route = route
			resultsChan <- target
			return
		}
	}
	target.RouteFound = false
	resultsChan <- target
}

// AttackCredentials attempts to guess the provided targets' credentials using the given
// dictionary or the default dictionary if none was provided by the user.
func AttackCredentials(c Curler, targets []Stream, credentials Credentials, timeout time.Duration, log bool) ([]Stream, error) {
	attacks := make(chan Stream)
	defer close(attacks)

	validate := v.New()
	for _, target := range targets {
		err := validate.Struct(target)
		if err != nil {
			return targets, errors.Wrap(err, "invalid targets")
		}

		go attackCameraCredentials(c, target, credentials, attacks, timeout, log)
	}

	attackResults := []Stream{}
	for range targets {
		attackResults = append(attackResults, <-attacks)
	}

	for _, result := range attackResults {
		if result.CredentialsFound == true {
			targets = replace(targets, result)
		}
	}

	return targets, nil
}

// AttackRoute attempts to guess the provided targets' streaming routes using the given
// dictionary or the default dictionary if none was provided by the user.
func AttackRoute(c Curler, targets []Stream, routes Routes, timeout time.Duration, log bool) ([]Stream, error) {
	attacks := make(chan Stream)
	defer close(attacks)

	validate := v.New()
	for _, target := range targets {
		err := validate.Struct(target)
		if err != nil {
			return targets, errors.Wrap(err, "invalid targets")
		}

		go attackCameraRoute(c, target, routes, attacks, timeout, log)
	}

	attackResults := []Stream{}
	for range targets {
		attackResults = append(attackResults, <-attacks)
	}

	for _, result := range attackResults {
		if result.RouteFound == true {
			targets = replace(targets, result)
		}
	}

	return targets, nil
}
